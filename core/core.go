package core

import (
	"context"
	"errors"
	"sync"

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
)

// initQueue lazily creates the shared task queue.
func initQueue() *queue.TaskQueue[Executable] {
	queueOnce.Do(func() {
		queueInstance = queue.NewTaskQueue[Executable]()
	})
	return queueInstance
}

type Executable interface {
	Type() tasktype.TaskType
	Title() string
	TaskID() string
	Execute(ctx context.Context) error
}

func worker(ctx context.Context, qe *queue.TaskQueue[Executable], semaphore chan struct{}) {
	logger := log.FromContext(ctx)
	execHooks := config.C().Hook.Exec
	for {
		semaphore <- struct{}{}
		qtask, err := qe.Get()
		if err != nil {
			logger.Error("Failed to get task from queue:", err)
			break // queue closed and empty
		}
		exe := qtask.Data
		taskCtx := qtask.Context()
		logger.Infof("Processing task: %s", exe.TaskID())
		if err := database.UpdateTaskStatus(taskCtx, exe.TaskID(), database.TaskStatusRunning, ""); err != nil {
			logger.Errorf("Failed to mark task %s as running: %v", exe.TaskID(), err)
		}
		taskevent.Emit(taskCtx, taskevent.Event{TaskID: exe.TaskID(), Phase: taskevent.PhaseStart})
		if err := ExecCommandString(taskCtx, execHooks.TaskBeforeStart); err != nil {
			logger.Errorf("Failed to execute before start hook for task %s: %v", exe.TaskID(), err)
		}
		err = exe.Execute(taskCtx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Infof("Task %s was canceled", exe.TaskID())
				if err := ExecCommandString(ctx, execHooks.TaskCancel); err != nil {
					logger.Errorf("Failed to execute cancel hook for task %s: %v", exe.TaskID(), err)
				}
			} else {
				logger.Errorf("Failed to execute task %s: %v", exe.TaskID(), err)
				if err := ExecCommandString(ctx, execHooks.TaskFail); err != nil {
					logger.Errorf("Failed to execute fail hook for task %s: %v", exe.TaskID(), err)
				}
			}
		} else {
			logger.Infof("Task %s completed successfully", exe.TaskID())
			if err := ExecCommandString(ctx, execHooks.TaskSuccess); err != nil {
				logger.Errorf("Failed to execute success hook for task %s: %v", exe.TaskID(), err)
			}
		}
		taskevent.Emit(taskCtx, taskevent.Event{TaskID: exe.TaskID(), Phase: taskevent.PhaseDone, Err: err})
		qe.Done(qtask.ID)
		if err := database.DeleteTask(ctx, exe.TaskID()); err != nil {
			logger.Errorf("Failed to delete persisted task %s: %v", exe.TaskID(), err)
		}
		<-semaphore
	}
}

func Run(ctx context.Context) {
	log.FromContext(ctx).Info("Start processing tasks...")
	semaphore := make(chan struct{}, config.C().Workers)
	q := initQueue()
	for range config.C().Workers {
		go worker(ctx, q, semaphore)
	}

}

// Close stops the queue and unblocks workers in Get.
func Close() {
	if q := initQueue(); q != nil {
		q.Close()
	}
}

func AddTask(ctx context.Context, task Executable) error {
	if err := persistTask(ctx, task); err != nil {
		log.FromContext(ctx).Errorf("Failed to persist task %s: %v", task.TaskID(), err)
	}
	return initQueue().Add(queue.NewTask(ctx, task.TaskID(), task.Title(), task))
}

func CancelTask(ctx context.Context, id string) error {
	err := queueInstance.CancelTask(id)
	if err != nil {
		return err
	}
	if err := database.DeleteTask(ctx, id); err != nil {
		log.FromContext(ctx).Errorf("Failed to delete persisted task %s: %v", id, err)
	}
	return nil
}

func GetLength(ctx context.Context) int {
	return queueInstance.ActiveLength()
}

func GetRunningTasks(ctx context.Context) []queue.TaskInfo {
	return queueInstance.RunningTasks()
}

func GetQueuedTasks(ctx context.Context) []queue.TaskInfo {
	return queueInstance.QueuedTasks()
}
