package core

import (
	"context"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/queue"
)

var queueInstance *queue.TaskQueue[Exectable]

type Exectable interface {
	Type() tasktype.TaskType
	TaskID() string
	Execute(ctx context.Context) error
}

func worker(ctx context.Context, qe *queue.TaskQueue[Exectable], semaphore chan struct{}) {
	logger := log.FromContext(ctx)
	execHooks := config.C().Hook.Exec
	for {
		semaphore <- struct{}{}
		qtask, err := qe.Get()
		if err != nil {
			logger.Error("Failed to get task from queue:", err)
			break // queue closed and empty
		}
		task := qtask.Data
		logger.Infof("Processing task: %s", task.TaskID())
		if err := ExecCommandString(qtask.Context(), execHooks.TaskBeforeStart); err != nil {
			logger.Errorf("Failed to execute before start hook for task %s: %v", task.TaskID(), err)
		}
		if err := task.Execute(qtask.Context()); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Infof("Task %s was canceled", task.TaskID())
				if err := ExecCommandString(ctx, execHooks.TaskCancel); err != nil {
					logger.Errorf("Failed to execute cancel hook for task %s: %v", task.TaskID(), err)
				}
			} else {
				logger.Errorf("Failed to execute task %s: %v", task.TaskID(), err)
				if err := ExecCommandString(ctx, execHooks.TaskFail); err != nil {
					logger.Errorf("Failed to execute fail hook for task %s: %v", task.TaskID(), err)
				}
			}
		} else {
			logger.Infof("Task %s completed successfully", task.TaskID())
			if err := ExecCommandString(ctx, execHooks.TaskSuccess); err != nil {
				logger.Errorf("Failed to execute success hook for task %s: %v", task.TaskID(), err)
			}
		}
		qe.Done(qtask.ID)
		<-semaphore
	}
}

func Run(ctx context.Context) {
	log.FromContext(ctx).Info("Start processing tasks...")
	semaphore := make(chan struct{}, config.C().Workers)
	if queueInstance == nil {
		queueInstance = queue.NewTaskQueue[Exectable]()
	}
	for range config.C().Workers {
		go worker(ctx, queueInstance, semaphore)
	}

}

func AddTask(ctx context.Context, task Exectable) error {
	return queueInstance.Add(queue.NewTask(ctx, task.TaskID(), task))
}

func CancelTask(ctx context.Context, id string) error {
	err := queueInstance.CancelTask(id)
	return err
}

func GetLength(ctx context.Context) int {
	if queueInstance == nil {
		return 0
	}
	return queueInstance.ActiveLength()
}
