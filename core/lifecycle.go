package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/upload"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/taskstatus"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/queue"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
)

var ErrTaskNotActive = errors.New("task is not active")

func runTask(ctx context.Context, qtask *queue.Task[Executable]) {
	logger := log.FromContext(ctx)
	task := qtask.Data
	taskCtx := storagetypes.WithPublishGuard(qtask.Context(), qtask.RunExclusive)
	recordCtx := context.WithoutCancel(taskCtx)
	isUpload := task.Type() == tasktype.TaskTypeUpload
	hooks := config.C().Hook.Exec

	started, err := database.TransitionTaskRecord(recordCtx, qtask.ID, []taskstatus.Status{taskstatus.Queued}, map[string]any{
		"status":     taskstatus.Running,
		"started_at": time.Now(),
	})
	if err != nil {
		logger.Errorf("Failed to start task %s: %v", qtask.ID, err)
		return
	}
	if !started {
		if isUpload {
			cleanupTerminalUpload(recordCtx, qtask.ID)
		}
		return
	}

	logger.Infof("Processing task: %s", qtask.ID)
	taskevent.Emit(taskCtx, taskevent.Event{TaskID: qtask.ID, Phase: taskevent.PhaseStart})
	if err := ExecCommandString(taskCtx, hooks.TaskBeforeStart); err != nil {
		logger.Errorf("Failed to execute before start hook for task %s: %v", qtask.ID, err)
	}
	err = taskCtx.Err()
	var items []ItemResult
	if err == nil {
		err = task.Execute(taskCtx)
		items = collectItemResults(task)
	}
	status := taskstatus.Completed
	finished := map[string]any{"finished_at": time.Now(), "error": ""}
	switch {
	case err != nil && taskCtx.Err() != nil:
		// A cancelled context alone means interruption, not a user cancellation.
		status = taskstatus.Queued
		finished["started_at"] = nil
		finished["finished_at"] = nil
	case err != nil:
		status = taskstatus.Failed
		finished["error"] = err.Error()
	case countItems(items, ItemStatusFailed) > 0:
		status = taskstatus.Partial
		finished["error"] = partialError(items)
	}
	var actualPath string
	if isUpload && status == taskstatus.Completed && len(items) == 1 {
		actualPath = items[0].Path
		finished["path"] = actualPath
	}
	finished["status"] = status
	var itemRecords []database.TaskItemRecord
	var metadata []database.TaskItemMetadata
	if status != taskstatus.Queued {
		itemRecords, metadata = taskItemRecords(items)
	}
	saved, dbErr := database.FinishTaskRecord(recordCtx, qtask.ID, []taskstatus.Status{taskstatus.Running}, finished, itemRecords, metadata)
	if dbErr != nil {
		logger.Errorf("Failed to finish task record for %s: %v", qtask.ID, dbErr)
		if isUpload {
			cleanupTerminalUpload(recordCtx, qtask.ID)
		}
		return
	}
	if !saved {
		current, err := database.GetTaskRecord(recordCtx, qtask.ID)
		if err != nil || current.Status != taskstatus.Cancelled.String() {
			return
		}
		status = taskstatus.Cancelled
		if isUpload {
			for i := range items {
				items[i].Status = ItemStatusCancelled
				items[i].Error = context.Canceled.Error()
			}
		}
		itemRecords, metadata = taskItemRecords(items)
		if len(itemRecords) > 0 {
			if _, err := database.FinishTaskRecord(recordCtx, qtask.ID, []taskstatus.Status{taskstatus.Cancelled},
				map[string]any{"status": taskstatus.Cancelled}, itemRecords, metadata); err != nil {
				logger.Errorf("Failed to store cancelled task results for %s: %v", qtask.ID, err)
			}
		}
	}
	if status == taskstatus.Queued {
		logger.Infof("Task %s interrupted; it will resume after restart", qtask.ID)
		return
	}
	if isUpload {
		cleanupTerminalUpload(recordCtx, qtask.ID)
	}

	event := taskevent.Event{TaskID: qtask.ID, Phase: taskevent.PhaseDone}
	var hook string
	switch status {
	case taskstatus.Cancelled:
		logger.Infof("Task %s was cancelled", qtask.ID)
		event.Cancelled = true
		hook = hooks.TaskCancel
	case taskstatus.Failed:
		logger.Errorf("Failed to execute task %s: %v", qtask.ID, err)
		event.Err = err
		hook = hooks.TaskFail
	case taskstatus.Partial:
		logger.Warnf("Task %s finished with failed files", qtask.ID)
		event.Partial = true
		hook = hooks.TaskFail
	case taskstatus.Completed:
		logger.Infof("Task %s completed successfully", qtask.ID)
		event.Path = actualPath
		hook = hooks.TaskSuccess
	}
	taskevent.Emit(taskCtx, event)
	if err := ExecCommandString(ctx, hook); err != nil {
		logger.Errorf("Failed to execute %s hook for task %s: %v", status, qtask.ID, err)
	}
}

func taskItemRecords(items []ItemResult) ([]database.TaskItemRecord, []database.TaskItemMetadata) {
	records := make([]database.TaskItemRecord, 0, len(items))
	var metadata []database.TaskItemMetadata
	for _, item := range items {
		records = append(records, database.TaskItemRecord{
			Index:        item.Index,
			Name:         item.Name,
			Path:         item.Path,
			StorageName:  item.StorageName,
			StorageID:    item.StorageID,
			Status:       string(item.Status),
			Error:        item.Error,
			Size:         item.Size,
			SourceChatID: item.SourceChatID,
			SourceMsgID:  item.SourceMessageID,
		})
		if item.MetadataJSON != "" {
			metadata = append(metadata, database.TaskItemMetadata{
				ItemIndex:       item.Index,
				Version:         item.MetadataVersion,
				SourceChatID:    item.SourceChatID,
				SourceMessageID: item.SourceMessageID,
				JSON:            item.MetadataJSON,
			})
		}
	}
	return records, metadata
}

func CancelTask(ctx context.Context, id string) error {
	recordCtx := context.WithoutCancel(ctx)
	record, err := database.GetTaskRecord(recordCtx, id)
	if err != nil {
		return fmt.Errorf("get task %s: %w", id, err)
	}
	q := initQueue()
	task, _ := q.GetTask(id)
	cancel := func() error {
		// Persist cancellation before cancelling the context to distinguish it
		// from a restart and prevent the worker from overwriting the status.
		updated, err := database.TransitionTaskRecord(recordCtx, id, []taskstatus.Status{taskstatus.Queued, taskstatus.Running}, map[string]any{
			"status":      taskstatus.Cancelled,
			"finished_at": time.Now(),
		})
		if err != nil {
			return err
		}
		if !updated && record.Status != taskstatus.Cancelled.String() {
			return ErrTaskNotActive
		}
		if task != nil {
			task.Cancel()
		}
		return nil
	}
	if task != nil {
		err = task.RunExclusive(cancel)
	} else {
		err = cancel()
		// Recovery may have enqueued the task while cancellation was persisted.
		if candidate, ok := q.GetTask(id); err == nil && ok {
			task = candidate
			_ = task.RunExclusive(func() error { task.Cancel(); return nil })
		}
	}
	if err != nil {
		return err
	}
	if task != nil {
		taskevent.Emit(task.Context(), taskevent.Event{TaskID: id, Phase: taskevent.PhaseDone, Cancelled: true})
	}
	if !q.IsRunning(id) {
		CleanupTaskArtifacts(recordCtx, record)
	}
	return nil
}

func cleanupTerminalUpload(ctx context.Context, taskID string) {
	if record, err := database.GetTaskRecord(ctx, taskID); err == nil && taskstatus.Status(record.Status).IsTerminal() {
		CleanupTaskArtifacts(ctx, record)
	}
}

func CleanupTaskArtifacts(ctx context.Context, record *database.TaskRecord) {
	if record.Type != tasktype.TaskTypeUpload.String() {
		return
	}
	var spec taskspec.Upload
	if err := json.Unmarshal([]byte(record.Params), &spec); err != nil {
		log.FromContext(ctx).Warnf("Failed to decode staging parameters for task %s: %v", record.TaskID, err)
		return
	}
	if err := upload.RemoveStaging(spec.StagingRoot, record.TaskID); err != nil {
		log.FromContext(ctx).Warnf("Failed to remove staging for task %s: %v", record.TaskID, err)
	}
}
