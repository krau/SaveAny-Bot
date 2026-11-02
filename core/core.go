package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/queue"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
)

var (
	queueOnce     sync.Once
	queueInstance *queue.TaskQueue[Executable]
	workerWG      sync.WaitGroup

	taskSinkFactoryMu sync.RWMutex
	taskSinkFactory   func(Executable) taskevent.Sink
)

// initQueue lazily creates the shared task queue.
func initQueue() *queue.TaskQueue[Executable] {
	queueOnce.Do(func() {
		queueInstance = queue.NewTaskQueue[Executable]()
	})
	return queueInstance
}

const (
	TaskSourceBot   = "bot"
	TaskSourceAPI   = "api"
	TaskSourceRetry = "retry"
)

type Executable interface {
	Type() tasktype.TaskType
	Title() string
	TaskID() string
	Execute(ctx context.Context) error
}

// DestinationInfo lets task records include the storage target for every
// creation path, including tasks started from Telegram handlers.
type DestinationInfo interface {
	StorageName() string
	StoragePath() string
}

// ParamSource exposes the creation parameters that a queued task needs to be
// rebuilt after a restart.
type ParamSource interface {
	TaskParams() (json.RawMessage, error)
}

func worker(ctx context.Context, qe *queue.TaskQueue[Executable], semaphore chan struct{}) {
	defer workerWG.Done()
	logger := log.FromContext(ctx)
	execHooks := config.C().Hook.Exec
	for {
		semaphore <- struct{}{}
		qtask, err := qe.Get()
		if err != nil {
			if !errors.Is(err, queue.ErrQueueClosed) {
				logger.Error("Failed to get task from queue:", err)
			}
			break // queue closed and empty
		}
		exe := qtask.Data
		taskCtx := qtask.Context()
		recordCtx := context.WithoutCancel(taskCtx)
		logger.Infof("Processing task: %s", exe.TaskID())
		taskevent.Emit(taskCtx, taskevent.Event{TaskID: exe.TaskID(), Phase: taskevent.PhaseStart})
		if err := ExecCommandString(taskCtx, execHooks.TaskBeforeStart); err != nil {
			logger.Errorf("Failed to execute before start hook for task %s: %v", exe.TaskID(), err)
		}
		startedAt := time.Now()
		if dbErr := database.UpdateTaskRecord(recordCtx, exe.TaskID(), map[string]any{
			"status":     "running",
			"started_at": startedAt,
		}); dbErr != nil {
			logger.Errorf("Failed to update task record for %s: %v", exe.TaskID(), dbErr)
		}
		err = exe.Execute(taskCtx)
		items := collectItemResults(exe)
		failedItems := countItems(items, ItemStatusFailed)
		status := "completed"
		if err != nil {
			if errors.Is(taskCtx.Err(), context.Canceled) {
				current, dbErr := database.GetTaskRecord(recordCtx, exe.TaskID())
				if dbErr == nil && current.Status == "cancelled" {
					status = "cancelled"
					logger.Infof("Task %s was canceled", exe.TaskID())
					if err := ExecCommandString(ctx, execHooks.TaskCancel); err != nil {
						logger.Errorf("Failed to execute cancel hook for task %s: %v", exe.TaskID(), err)
					}
				} else {
					// Process shutdown cancels task contexts; leave the record queued
					// so the next start can resume it from persisted cache.
					status = "queued"
					logger.Infof("Task %s interrupted; it will resume after restart", exe.TaskID())
				}
			} else {
				status = "failed"
				logger.Errorf("Failed to execute task %s: %v", exe.TaskID(), err)
				if err := ExecCommandString(ctx, execHooks.TaskFail); err != nil {
					logger.Errorf("Failed to execute fail hook for task %s: %v", exe.TaskID(), err)
				}
			}
		} else if failedItems > 0 {
			status = "partial"
			logger.Warnf("Task %s finished with %d failed files", exe.TaskID(), failedItems)
			if err := ExecCommandString(ctx, execHooks.TaskFail); err != nil {
				logger.Errorf("Failed to execute fail hook for task %s: %v", exe.TaskID(), err)
			}
		} else {
			logger.Infof("Task %s completed successfully", exe.TaskID())
			if err := ExecCommandString(ctx, execHooks.TaskSuccess); err != nil {
				logger.Errorf("Failed to execute success hook for task %s: %v", exe.TaskID(), err)
			}
		}
		if len(items) > 0 && status != "queued" {
			records := make([]database.TaskItemRecord, 0, len(items))
			for _, item := range items {
				records = append(records, database.TaskItemRecord{
					Index:        item.Index,
					Name:         item.Name,
					Path:         item.Path,
					StorageName:  item.StorageName,
					Status:       string(item.Status),
					Error:        item.Error,
					Size:         item.Size,
					SourceChatID: item.SourceChatID,
					SourceMsgID:  item.SourceMessageID,
				})
			}
			if dbErr := database.ReplaceTaskItems(recordCtx, exe.TaskID(), records); dbErr != nil {
				logger.Errorf("Failed to store task items for %s: %v", exe.TaskID(), dbErr)
			}
		}
		recordFields := map[string]any{"status": status}
		if status == "queued" {
			recordFields["started_at"] = nil
			recordFields["finished_at"] = nil
			recordFields["error"] = ""
		} else {
			recordFields["finished_at"] = time.Now()
		}
		if status == "partial" {
			recordFields["error"] = partialError(items)
		} else if err != nil && status != "cancelled" {
			recordFields["error"] = err.Error()
		}
		if dbErr := database.UpdateTaskRecord(recordCtx, exe.TaskID(), recordFields); dbErr != nil {
			logger.Errorf("Failed to finish task record for %s: %v", exe.TaskID(), dbErr)
		}
		taskevent.Emit(taskCtx, taskevent.Event{
			TaskID:    exe.TaskID(),
			Phase:     taskevent.PhaseDone,
			Err:       err,
			Partial:   failedItems > 0 && err == nil,
			Cancelled: status == "cancelled",
		})
		qe.Done(qtask.ID)
		<-semaphore
	}
}

func Run(ctx context.Context) {
	log.FromContext(ctx).Info("Start processing tasks...")
	semaphore := make(chan struct{}, config.C().Workers)
	q := initQueue()
	workers := config.C().Workers
	workerWG.Add(workers)
	for range workers {
		go worker(ctx, q, semaphore)
	}
}

// Close stops the queue and unblocks workers in Get.
func Close() {
	if q := initQueue(); q != nil {
		q.Close()
	}
}

// WaitWorkers waits up to timeout for all worker goroutines to exit.
func WaitWorkers(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		workerWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func AddTask(ctx context.Context, task Executable) error {
	return AddTaskWithSource(ctx, task, TaskSourceBot)
}

// SetTaskSinkFactory installs the optional observer used for tasks started
// outside the API. The API task path registers its own sink with webhook data.
func SetTaskSinkFactory(factory func(Executable) taskevent.Sink) {
	taskSinkFactoryMu.Lock()
	taskSinkFactory = factory
	taskSinkFactoryMu.Unlock()
}

func taskSinkFor(task Executable, source string) taskevent.Sink {
	if source == TaskSourceAPI {
		return nil
	}
	taskSinkFactoryMu.RLock()
	factory := taskSinkFactory
	taskSinkFactoryMu.RUnlock()
	if factory == nil {
		return nil
	}
	return factory(task)
}

// AddTaskWithSource records the task before enqueueing it so every execution
// has a matching database row.
func AddTaskWithSource(ctx context.Context, task Executable, source string) error {
	logger := log.FromContext(ctx)
	sink := taskSinkFor(task, source)
	if sink != nil {
		ctx = taskevent.WithSink(ctx, sink)
	}
	record := &database.TaskRecord{
		TaskID: task.TaskID(),
		Type:   task.Type().String(),
		Title:  task.Title(),
		Status: "queued",
		Source: source,
	}
	if destination, ok := task.(DestinationInfo); ok {
		record.StorageName = destination.StorageName()
		record.StorageID = config.C().GetStorageIDByName(record.StorageName)
		record.Path = destination.StoragePath()
	}
	if source, ok := task.(ParamSource); ok {
		params, err := source.TaskParams()
		if err != nil {
			logger.Warnf("Failed to serialize params for task %s: %v", task.TaskID(), err)
		} else {
			record.Params = string(params)
		}
	}
	if err := database.CreateTaskRecord(ctx, record); err != nil {
		logger.Errorf("Failed to create task record for %s: %v", task.TaskID(), err)
	}
	if err := initQueue().Add(queue.NewTask(ctx, task.TaskID(), task.Title(), task)); err != nil {
		if sink != nil {
			sink.Emit(taskevent.Event{TaskID: task.TaskID(), Phase: taskevent.PhaseDone, Err: err})
		}
		if dbErr := database.DeleteTaskRecord(ctx, task.TaskID()); dbErr != nil {
			logger.Errorf("Failed to remove task record for %s: %v", task.TaskID(), dbErr)
		}
		return err
	}
	return nil
}

func EnrichTaskRecord(ctx context.Context, taskID, storageName, storagePath string) error {
	return database.UpdateTaskRecord(ctx, taskID, map[string]any{
		"storage_name": storageName,
		"storage_id":   config.C().GetStorageIDByName(storageName),
		"path":         storagePath,
	})
}

// IsTaskRunning reports whether the task is still being executed, including
// the window between cancellation and the worker persisting its results.
func IsTaskRunning(id string) bool {
	return initQueue().IsRunning(id)
}

// HasTask reports whether the task is known to the queue, queued or running.
func HasTask(id string) bool {
	return initQueue().Has(id)
}

// EnqueueExisting adds a rebuilt task without creating another record. The
// progress sink factory is attached so the web console still tracks it.
func EnqueueExisting(ctx context.Context, task Executable) error {
	if sink := taskSinkFor(task, TaskSourceBot); sink != nil {
		ctx = taskevent.WithSink(ctx, sink)
	}
	return initQueue().Add(queue.NewTask(ctx, task.TaskID(), task.Title(), task))
}

func CancelTask(ctx context.Context, id string) error {
	q := initQueue()
	if q.Has(id) {
		task, _ := q.GetTask(id)
		// Persist the cancellation before canceling the context so the worker
		// can distinguish it from a process shutdown and does not requeue it.
		if dbErr := database.UpdateTaskRecord(context.WithoutCancel(ctx), id, map[string]any{
			"status":      "cancelled",
			"finished_at": time.Now(),
		}); dbErr != nil {
			log.FromContext(ctx).Errorf("Failed to cancel task record for %s: %v", id, dbErr)
		}
		err := q.CancelTask(id)
		if err == nil && task != nil {
			taskevent.Emit(task.Context(), taskevent.Event{
				TaskID:    id,
				Phase:     taskevent.PhaseDone,
				Cancelled: true,
			})
		}
		return err
	}
	// A queued record may not be in the queue yet while restart recovery waits
	// for a Telegram client.
	record, err := database.GetTaskRecord(ctx, id)
	if err != nil {
		return fmt.Errorf("task %s does not exist", id)
	}
	if isTerminalTaskStatus(record.Status) {
		return fmt.Errorf("task %s is not active", id)
	}
	if dbErr := database.UpdateTaskRecord(context.WithoutCancel(ctx), id, map[string]any{
		"status":      "cancelled",
		"finished_at": time.Now(),
	}); dbErr != nil {
		return dbErr
	}
	if q.Has(id) {
		return q.CancelTask(id)
	}
	return nil
}

func isTerminalTaskStatus(status string) bool {
	switch status {
	case "completed", "partial", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func GetLength(ctx context.Context) int {
	if queueInstance == nil {
		return 0
	}
	return queueInstance.ActiveLength()
}

// ActiveTaskCount includes queued and running tasks regardless of cancellation.
func ActiveTaskCount() int {
	if queueInstance == nil {
		return 0
	}
	return queueInstance.ActiveCount()
}

func GetRunningTasks(ctx context.Context) []queue.TaskInfo {
	return queueInstance.RunningTasks()
}

func GetQueuedTasks(ctx context.Context) []queue.TaskInfo {
	return queueInstance.QueuedTasks()
}
