package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func processPendingTask(task *types.Task) error {
	logger.L.Debugf("Start processing task: %s", task.String())

	os.MkdirAll(config.Cfg.Temp.BasePath, os.ModePerm)

	logger.L.Debugf("Start downloading file: %s", task.String())

	task.Ctx.(*ext.Context).EditMessage(task.ChatID, &tg.MessagesEditMessageRequest{
		Message: "开始下载文件...",
		ID:      task.ReplyMessageID,
	})

	readCloser, err := NewTelegramReader(task.Ctx, bot.Client, task.File.Location, 0, task.File.FileSize-1, task.File.FileSize)
	if err != nil {
		return fmt.Errorf("Failed to create reader: %w", err)
	}
	defer readCloser.Close()

	dest, err := os.Create(common.GetCacheFilePath(task.FileName()))
	if err != nil {
		return fmt.Errorf("Failed to create file: %w", err)
	}
	logger.L.Debug("Created file: ", dest.Name())
	defer dest.Close()

	if _, err := io.CopyN(dest, readCloser, task.File.FileSize); err != nil {
		return fmt.Errorf("Failed to download file: %w", err)
	}

	defer func() {
		if config.Cfg.Temp.CacheTTL > 0 {
			common.RmFileAfter(dest.Name(), time.Duration(config.Cfg.Temp.CacheTTL)*time.Second)
		} else {
			if err := os.Remove(dest.Name()); err != nil {
				logger.L.Errorf("Failed to purge file: %s", err)
			}
		}
	}()

	if task.StoragePath == "" {
		task.StoragePath = task.File.FileName
	}

	task.Ctx.(*ext.Context).EditMessage(task.ChatID, &tg.MessagesEditMessageRequest{
		Message: "下载完成, 正在转存文件...",
		ID:      task.ReplyMessageID,
	})

	if err := storage.Save(task.Storage, task.Ctx, dest.Name(), task.StoragePath); err != nil {
		return fmt.Errorf("Failed to save file: %w", err)
	}
	return nil
}

func worker(queue *queue.TaskQueue, semaphore chan struct{}) {
	for {
		semaphore <- struct{}{}
		task := queue.GetTask()
		logger.L.Debugf("Got task: %s", task.String())

		switch task.Status {
		case types.Pending:
			logger.L.Infof("Processing task: %s", task.String())
			if err := processPendingTask(&task); err != nil {
				logger.L.Errorf("Failed to do task: %s", err)
				task.Error = err
				if errors.Is(err, context.Canceled) {
					logger.L.Debugf("Task canceled: %s", task.String())
					task.Status = types.Canceled
				} else {
					task.Status = types.Failed
				}
			} else {
				task.Status = types.Succeeded
			}
			queue.AddTask(task)
		case types.Succeeded:
			logger.L.Infof("Task succeeded: %s", task.String())
			task.Ctx.(*ext.Context).EditMessage(task.ChatID, &tg.MessagesEditMessageRequest{
				Message: "文件保存成功",
				ID:      task.ReplyMessageID,
			})
		case types.Failed:
			logger.L.Errorf("Task failed: %s", task.String())
			task.Ctx.(*ext.Context).EditMessage(task.ChatID, &tg.MessagesEditMessageRequest{
				Message: "文件保存失败",
				ID:      task.ReplyMessageID,
			})
		case types.Canceled:
			logger.L.Infof("Task canceled: %s", task.String())
		default:
			logger.L.Errorf("Unknown task status: %s", task.Status)
		}
		<-semaphore
		logger.L.Debugf("Task done: %s", task.String())
	}
}

func Run() {
	logger.L.Info("Start processing tasks...")
	semaphore := make(chan struct{}, config.Cfg.Workers)
	for i := 0; i < config.Cfg.Workers; i++ {
		go worker(queue.Queue, semaphore)
	}

}
