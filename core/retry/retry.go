// Package retry re-creates a task from its persisted item records, so files
// that failed (for example because the disk was full) can be downloaded again
// without resubmitting the original message.
package retry

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/batchtfile"
	"github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/taskstatus"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	tfilepkg "github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/ext"
	"github.com/rs/xid"
)

var (
	ErrTaskActive        = errors.New("task is still active")
	ErrNothingToRetry    = errors.New("task has no failed files to retry")
	ErrUnsupportedType   = errors.New("retry is only supported for telegram file tasks")
	ErrStorageMissing    = errors.New("target storage is no longer available")
	ErrSourceUnavailable = errors.New("failed file has no source message to retry")
	ErrClientUnavailable = errors.New("no telegram client available")
)

// FailedTask enqueues a new task containing the failed (or unfinished) files
// of taskID and returns the new task ID. Messages are fetched fresh so their
// file references are current. extCtxs are tried in order (bot first, then
// userbot) to match the client that can see the source chat.
func FailedTask(ctx context.Context, extCtxs []*ext.Context, taskID string) (string, int, error) {
	record, err := database.GetTaskRecord(ctx, taskID)
	if err != nil {
		return "", 0, fmt.Errorf("task not found: %w", err)
	}
	if core.IsTaskRunning(taskID) {
		return "", 0, ErrTaskActive
	}
	if !isTerminalStatus(record.Status) {
		return "", 0, ErrTaskActive
	}
	if record.Type != tasktype.TaskTypeTgfiles.String() {
		return "", 0, ErrUnsupportedType
	}
	items, err := database.ListTaskItems(ctx, taskID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to load task items: %w", err)
	}
	selected := retryableItems(items)
	if len(selected) == 0 {
		return "", 0, ErrNothingToRetry
	}
	candidates := make([]*ext.Context, 0, len(extCtxs))
	for _, candidate := range extCtxs {
		if candidate != nil {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return "", 0, ErrClientUnavailable
	}

	elems := make([]batchtfile.TaskElement, 0, len(selected))
	var taskExt *ext.Context
	for _, item := range selected {
		if item.SourceChatID == 0 || item.SourceMsgID == 0 {
			return "", 0, fmt.Errorf("%w: %s", ErrSourceUnavailable, item.Name)
		}
		msg, clientCtx, err := tgutil.GetMessageByIDFreshAny(candidates, item.SourceChatID, item.SourceMsgID)
		if err != nil {
			return "", 0, fmt.Errorf("%w: %s: %v", ErrSourceUnavailable, item.Name, err)
		}
		if taskExt == nil {
			taskExt = clientCtx
		}
		if msg.Media == nil {
			return "", 0, fmt.Errorf("%w: %s", ErrSourceUnavailable, item.Name)
		}
		stor, err := itemStorage(ctx, item, record)
		if err != nil {
			return "", 0, err
		}
		file, err := tfilepkg.FromMediaMessage(msg.Media, clientCtx.Raw, msg)
		if err != nil {
			return "", 0, fmt.Errorf("failed to rebuild file %s: %w", item.Name, err)
		}
		file.SetName(item.Name)
		target := item.Path
		if target == "" {
			target = path.Join(record.Path, item.Name)
		}
		elem, err := batchtfile.NewTaskElement(stor, target, file)
		if err != nil {
			return "", 0, fmt.Errorf("failed to create task element for %s: %w", item.Name, err)
		}
		elems = append(elems, *elem)
	}

	newID := xid.New().String()
	// A failed item may have left a partial file at its destination; the retry
	// must replace it instead of writing a renamed duplicate.
	taskCtx := storage.WithOverwrite(tgutil.ExtWithContext(ctx, taskExt))
	var task core.Executable
	if len(elems) == 1 {
		single, err := tfile.NewTGFileTask(newID, taskCtx, elems[0].File, elems[0].Storage, elems[0].Path, nil)
		if err != nil {
			return "", 0, fmt.Errorf("failed to create retry task: %w", err)
		}
		task = single
	} else {
		task = batchtfile.NewBatchTGFileTask(newID, taskCtx, elems, nil, true)
	}
	if err := core.AddTaskWithSource(taskCtx, task, core.TaskSourceRetry); err != nil {
		return "", 0, fmt.Errorf("failed to enqueue retry task: %w", err)
	}
	return newID, len(elems), nil
}

func itemStorage(ctx context.Context, item database.TaskItemRecord, record *database.TaskRecord) (storage.Storage, error) {
	id := item.StorageID
	if id == 0 {
		id = record.StorageID
	}
	if id != 0 {
		if stor, err := storage.GetStorageByID(ctx, id); err == nil {
			return stor, nil
		}
	}
	name := item.StorageName
	if name == "" {
		name = record.StorageName
	}
	stor, ok := storage.GetStorage(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrStorageMissing, name)
	}
	return stor, nil
}

// retryableItems prefers failed files; if a task failed before per-item
// outcomes were recorded, unfinished items are retried instead.
func retryableItems(items []database.TaskItemRecord) []database.TaskItemRecord {
	failed := make([]database.TaskItemRecord, 0, len(items))
	for _, item := range items {
		if item.Status == string(core.ItemStatusFailed) {
			failed = append(failed, item)
		}
	}
	if len(failed) > 0 {
		return failed
	}
	for _, item := range items {
		switch item.Status {
		case string(core.ItemStatusCompleted), string(core.ItemStatusSkipped):
			continue
		default:
			failed = append(failed, item)
		}
	}
	return failed
}

func isTerminalStatus(status string) bool {
	return taskstatus.Status(status).IsTerminal()
}
