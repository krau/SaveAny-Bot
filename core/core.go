package core

import (
	"context"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/queue"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
)

var queueInstance *queue.TaskQueue[Executable]

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
		<-semaphore
	}
}

func Run(ctx context.Context) {
	log.FromContext(ctx).Info("Start processing tasks...")
	semaphore := make(chan struct{}, config.C().Workers)
	if queueInstance == nil {
		queueInstance = queue.NewTaskQueue[Executable]()
	}
	for range config.C().Workers {
		go worker(ctx, queueInstance, semaphore)
	}

}

func AddTask(ctx context.Context, task Executable) error {
	return queueInstance.Add(queue.NewTask(ctx, task.TaskID(), task.Title(), task))
}

func CancelTask(ctx context.Context, id string) error {
	err := queueInstance.CancelTask(id)
	return err
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
