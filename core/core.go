package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/types"
)

func worker(queue *queue.TaskQueue, semaphore chan struct{}) {
	for {
		semaphore <- struct{}{}
		task := queue.GetTask()
		logger.L.Debugf("Got task: %s", task.String())

		switch task.Status {
		case types.Pending:
			logger.L.Infof("Processing task: %s", task.String())
			if err := processPendingTask(task); err != nil {
				task.Error = err
				if errors.Is(err, context.Canceled) {
					task.Status = types.Canceled
				} else {
					logger.L.Errorf("Failed to do task: %s", err)
					task.Status = types.Failed
				}
			} else {
				task.Status = types.Succeeded
			}
			queue.AddTask(task)
		case types.Succeeded:
			logger.L.Infof("Task succeeded: %s", task.String())
			extCtx, ok := task.Ctx.(*ext.Context)
			if !ok {
				logger.L.Errorf("Context is not *ext.Context: %T", task.Ctx)
			} else {
				extCtx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
					Message: fmt.Sprintf("文件保存成功\n [%s]: %s", task.StorageName, task.StoragePath),
					ID:      task.ReplyMessageID,
				})
			}
		case types.Failed:
			logger.L.Errorf("Task failed: %s", task.String())
			extCtx, ok := task.Ctx.(*ext.Context)
			if !ok {
				logger.L.Errorf("Context is not *ext.Context: %T", task.Ctx)
			} else {
				extCtx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
					Message: "文件保存失败\n" + task.Error.Error(),
					ID:      task.ReplyMessageID,
				})
			}
		case types.Canceled:
			logger.L.Infof("Task canceled: %s", task.String())
			extCtx, ok := task.Ctx.(*ext.Context)
			if !ok {
				logger.L.Errorf("Context is not *ext.Context: %T", task.Ctx)
			} else {
				extCtx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
					Message: "任务已取消",
					ID:      task.ReplyMessageID,
				})
			}
		default:
			logger.L.Errorf("Unknown task status: %s", task.Status)
		}
		<-semaphore
		logger.L.Debugf("Task done: %s; status: %s", task.String(), task.Status)
		queue.DoneTask(task)
	}
}

func Run() {
	logger.L.Info("Start processing tasks...")
	semaphore := make(chan struct{}, config.Cfg.Workers)
	for i := 0; i < config.Cfg.Workers; i++ {
		go worker(queue.Queue, semaphore)
	}

}
