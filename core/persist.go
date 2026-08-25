package core

import (
	"context"
	"sync"
	"time"

	"fmt"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/downloader"

	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
)

// TaskCodec serializes and rebuilds a task from its persisted payload.
// Task types without a registered codec are dropped with a warning on
// recovery instead of being silently re-enqueued.
type TaskCodec interface {
	Marshal(task Executable) ([]byte, error)
	Unmarshal(payload []byte) (Executable, error)
}

var (
	taskCodecsMu sync.RWMutex
	taskCodecs   = make(map[tasktype.TaskType]TaskCodec)

	dlerMu       sync.RWMutex
	dlerProvider func() downloader.Client
)

func RegisterTaskCodec(t tasktype.TaskType, codec TaskCodec) {
	taskCodecsMu.Lock()
	defer taskCodecsMu.Unlock()
	taskCodecs[t] = codec
}

func TaskCodecFor(t tasktype.TaskType) (TaskCodec, bool) {
	taskCodecsMu.RLock()
	defer taskCodecsMu.RUnlock()
	codec, ok := taskCodecs[t]
	return codec, ok
}

// SetDownloaderProvider registers the download client factory used to
// rebuild tfile.TGFile values when recovering tasks.
func SetDownloaderProvider(f func() downloader.Client) {
	dlerMu.Lock()
	defer dlerMu.Unlock()
	dlerProvider = f
}

// DownloaderClient returns the registered download client, or nil.
func DownloaderClient() downloader.Client {
	dlerMu.RLock()
	defer dlerMu.RUnlock()
	if dlerProvider == nil {
		return nil
	}
	return dlerProvider()
}

func persistTask(ctx context.Context, task Executable) error {
	codec, ok := TaskCodecFor(task.Type())
	if !ok {
		return nil
	}
	payload, err := codec.Marshal(task)
	if err != nil {
		return err
	}
	return database.UpsertTask(ctx, &database.Task{
		ID:      task.TaskID(),
		Type:    string(task.Type()),
		Payload: payload,
		Status:  string(database.TaskStatusQueued),
	})
}

// UpdateTaskPayload atomically mutates the persisted payload of a running
// task (e.g. recording per-element upload progress for recovery).
func UpdateTaskPayload(ctx context.Context, id string, mutate func(payload []byte) ([]byte, error)) error {
	row, err := database.GetTask(ctx, id)
	if err != nil {
		return err
	}
	updated, err := mutate(row.Payload)
	if err != nil {
		return fmt.Errorf("mutate payload: %w", err)
	}
	return database.UpdateTaskPayload(ctx, id, updated)
}

// RecoverTasks re-enqueues tasks that were unfinished when the process last
// exited. Must be called after storages are loaded and before Run. Tasks
// that cannot be recovered are marked failed and kept for visibility.
func RecoverTasks(ctx context.Context) {
	logger := log.FromContext(ctx)
	if err := database.DeleteStaleFailedTasks(ctx, 24*time.Hour); err != nil {
		logger.Warnf("Failed to clean stale failed tasks: %v", err)
	}
	tasks, err := database.GetUnfinishedTasks(ctx)
	if err != nil {
		logger.Errorf("Failed to load unfinished tasks: %v", err)
		return
	}
	for _, t := range tasks {
		codec, ok := TaskCodecFor(tasktype.TaskType(t.Type))
		if !ok {
			logger.Warnf("Task %s (type %s) cannot be recovered: no codec registered", t.ID, t.Type)
			markRecoverFailed(ctx, t, "no codec registered")
			continue
		}
		task, err := codec.Unmarshal(t.Payload)
		if err != nil {
			logger.Errorf("Task %s cannot be recovered: failed to rebuild: %v", t.ID, err)
			markRecoverFailed(ctx, t, err.Error())
			continue
		}
		if initQueue().Contains(task.TaskID()) {
			// Already live in the queue (e.g. submitted via API during
			// startup); keep the row as-is.
			logger.Infof("Task %s already queued, keeping row", t.ID)
			continue
		}
		if err := AddTask(ctx, task); err != nil {
			logger.Errorf("Task %s cannot be recovered: failed to re-enqueue: %v", t.ID, err)
			markRecoverFailed(ctx, t, err.Error())
			continue
		}
		// Upsert cleared the original creation time; restore it so
		// GetUnfinishedTasks ordering stays stable across restarts.
		if err := database.RestoreTaskCreatedAt(ctx, t.ID, t.CreatedAt); err != nil {
			logger.Warnf("Failed to restore created_at for task %s: %v", t.ID, err)
		}
		logger.Infof("Recovered task %s (%s)", t.ID, t.Type)
	}
}

func markRecoverFailed(ctx context.Context, t database.Task, reason string) {
	if err := database.UpdateTaskStatus(ctx, t.ID, database.TaskStatusFailed, reason); err != nil {
		log.FromContext(ctx).Errorf("Failed to mark task %s as failed: %v", t.ID, err)
	}
}
