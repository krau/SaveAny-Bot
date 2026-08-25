package core

import (
	"context"
	"sync"

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

// RecoverTasks re-enqueues tasks that were unfinished when the process last
// exited. Must be called after storages are loaded and before Run. Tasks of
// types without a registered codec are dropped with a warning.
func RecoverTasks(ctx context.Context) {
	logger := log.FromContext(ctx)
	tasks, err := database.GetUnfinishedTasks(ctx)
	if err != nil {
		logger.Errorf("Failed to load unfinished tasks: %v", err)
		return
	}
	for _, t := range tasks {
		codec, ok := TaskCodecFor(tasktype.TaskType(t.Type))
		if !ok {
			logger.Warnf("Dropping unrecoverable task %s (type %s): no codec registered", t.ID, t.Type)
			if err := database.DeleteTask(ctx, t.ID); err != nil {
				logger.Errorf("Failed to delete task %s: %v", t.ID, err)
			}
			continue
		}
		task, err := codec.Unmarshal(t.Payload)
		if err != nil {
			logger.Errorf("Dropping task %s: failed to rebuild: %v", t.ID, err)
			if err := database.DeleteTask(ctx, t.ID); err != nil {
				logger.Errorf("Failed to delete task %s: %v", t.ID, err)
			}
			continue
		}
		if err := AddTask(ctx, task); err != nil {
			logger.Errorf("Dropping task %s: failed to re-enqueue: %v", t.ID, err)
			if err := database.DeleteTask(ctx, t.ID); err != nil {
				logger.Errorf("Failed to delete task %s: %v", t.ID, err)
			}
			continue
		}
		logger.Infof("Recovered task %s (%s)", t.ID, t.Type)
	}
}
