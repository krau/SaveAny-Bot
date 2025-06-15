package core

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/queue"
)

var queueInstance *queue.TaskQueue[Exectable]

type Exectable interface {
	TaskID() string
	Execute(ctx context.Context) error
}

func worker(ctx context.Context, qe *queue.TaskQueue[Exectable], semaphore chan struct{}) {
	for {
		semaphore <- struct{}{}
		qtask, err := qe.Get()
		if err != nil {
			break // queue closed and empty
		}
		log.FromContext(ctx).Infof("Processing task: %s", qtask.ID)
		task := qtask.Data
		if err := task.Execute(qtask.Context()); err != nil {
			log.FromContext(ctx).Errorf("Failed to execute task %s: %v", qtask.ID, err)
		} else {
			log.FromContext(ctx).Infof("Task %s completed successfully", qtask.ID)
		}
		qe.Done(qtask.ID)
		<-semaphore
	}
}

func Run(ctx context.Context) {
	log.FromContext(ctx).Info("Start processing tasks...")
	semaphore := make(chan struct{}, config.Cfg.Workers)
	if queueInstance == nil {
		queueInstance = queue.NewTaskQueue[Exectable]()
	}
	for range config.Cfg.Workers {
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
