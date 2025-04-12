package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/types"
)

var Downloader *downloader.Downloader

func init() {
	Downloader = downloader.NewDownloader().WithPartSize(1024 * 1024)
}

func worker(queue *queue.TaskQueue, semaphore chan struct{}) {
	for {
		semaphore <- struct{}{}
		task := queue.GetTask()
		common.Log.Debugf("Got task: %s", task.String())

		switch task.Status {
		case types.Pending:
			common.Log.Infof("Processing task: %s", task.String())
			if err := processPendingTask(task); err != nil {
				task.Error = err
				if errors.Is(err, context.Canceled) {
					task.Status = types.Canceled
				} else {
					common.Log.Errorf("Failed to do task: %s", err)
					task.Status = types.Failed
				}
			} else {
				task.Status = types.Succeeded
			}
			queue.AddTask(task)
		case types.Succeeded:
			common.Log.Infof("Task succeeded: %s", task.String())
			extCtx, ok := task.Ctx.(*ext.Context)
			if !ok {
				common.Log.Errorf("Context is not *ext.Context: %T", task.Ctx)
			} else if task.ReplyMessageID != 0 {
				extCtx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
					Message: fmt.Sprintf("文件保存成功\n [%s]: %s", task.StorageName, task.StoragePath),
					ID:      task.ReplyMessageID,
				})
			}
		case types.Failed:
			common.Log.Errorf("Task failed: %s", task.String())
			extCtx, ok := task.Ctx.(*ext.Context)
			if !ok {
				common.Log.Errorf("Context is not *ext.Context: %T", task.Ctx)
			} else if task.ReplyMessageID != 0 {
				extCtx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
					Message: "文件保存失败\n" + task.Error.Error(),
					ID:      task.ReplyMessageID,
				})
			}
		case types.Canceled:
			common.Log.Infof("Task canceled: %s", task.String())
			extCtx, ok := task.Ctx.(*ext.Context)
			if !ok {
				common.Log.Errorf("Context is not *ext.Context: %T", task.Ctx)
			} else if task.ReplyMessageID != 0 {
				extCtx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
					Message: "任务已取消",
					ID:      task.ReplyMessageID,
				})
			}
		default:
			common.Log.Errorf("Unknown task status: %s", task.Status)
		}
		<-semaphore
		common.Log.Debugf("Task done: %s; status: %s", task.String(), task.Status)
		queue.DoneTask(task)
	}
}

func Run() {
	common.Log.Info("Start processing tasks...")
	semaphore := make(chan struct{}, config.Cfg.Workers)
	for i := 0; i < config.Cfg.Workers; i++ {
		go worker(queue.Queue, semaphore)
	}

}
