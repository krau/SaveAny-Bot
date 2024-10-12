package core

import (
	"context"
	"errors"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func processPendingTask(task types.Task) error {
	message, err := bot.Client.GetMessageByID(task.ChatID, task.MessageID)
	if err != nil {
		return err
	}
	logger.L.Debugf("Start downloading file: %s", task.FileName)
	dest, err := message.Download(&telegram.DownloadOptions{
		FileName: common.GetCacheFilePath(task.FileName),
		// ProgressCallback: func(totalBytes, downloadedBytes int64) {},
	})
	if err != nil {
		return err
	}

	defer func() {
		if config.Cfg.Temp.CacheTTL > 0 {
			common.PurgeFileAfter(dest, time.Duration(config.Cfg.Temp.CacheTTL)*time.Second)
		} else {
			if err := common.PurgeFile(dest); err != nil {
				logger.L.Errorf("Failed to purge file: %s", err)
			}
		}
	}()
	if task.StoragePath == "" {
		task.StoragePath = task.FileName
	}

	if err := storage.Save(task.Storage, task.Ctx, dest, task.StoragePath); err != nil {
		return err
	}
	return nil
}

func worker(queue *queue.TaskQueue, semaphore chan struct{}) {
	for {
		semaphore <- struct{}{}
		task := queue.GetTask()
		logger.L.Debugf("Got task: %s", task.FileName)

		switch task.Status {
		case types.Pending:
			logger.L.Infof("Processing task: %s", task.String())
			if err := processPendingTask(task); err != nil {
				logger.L.Errorf("Failed to do task: %s", err)
				task.Error = err
				if errors.Is(err, context.Canceled) {
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
		case types.Failed:
			logger.L.Errorf("Task failed: %s", task.String())
		case types.Canceled:
			logger.L.Infof("Task canceled: %s", task.String())
		default:
			logger.L.Errorf("Unknown task status: %s", task.Status)
		}
		<-semaphore
		logger.L.Debugf("Task done: %s", task.FileName)
	}
}

func Run() {
	logger.L.Info("Start processing tasks...")
	semaphore := make(chan struct{}, 3)
	for i := 0; i < 3; i++ {
		go worker(queue.Queue, semaphore)
	}
}
