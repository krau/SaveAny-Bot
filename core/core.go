package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/imroc/req/v3"
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

func processPendingTask(task types.Task) error {
	fileRecord, err := dao.GetReceivedFileByFileID(task.FileID)
	if err != nil {
		return err
	}
	fileRecord.Processing = true
	if err := dao.UpdateReceivedFile(fileRecord); err != nil {
		return err
	}

	_, err = common.ReqClient.R().SetOutputFile(task.FileName).SetDownloadCallbackWithInterval(func(info req.DownloadInfo) {
		if info.Response == nil || info.Response.Response == nil || info.Response.Response.StatusCode != 200 {
			return
		}
		process := fmt.Sprintf("%.2f%%", float64(info.DownloadedSize)/float64(info.Response.ContentLength)*100.0)
		bot.Bot.EditMessageText(&telego.EditMessageTextParams{
			ChatID:    telegoutil.ID(task.ChatID),
			MessageID: task.ReplyMessageID,
			Text:      "正在下载文件: " + process,
		})
	}, time.Second*time.Duration(3*func() int {
		if queue.Len() > 0 {
			return queue.Len()
		}
		return 1
	}())).Get(bot.FileDownloadURL(task.FilePath))
	if err != nil {
		return err
	}

	bot.Bot.EditMessageText(&telego.EditMessageTextParams{
		ChatID:    telegoutil.ID(task.ChatID),
		MessageID: task.ReplyMessageID,
		Text:      "下载完成, 正在转存...",
	})

	defer func() {
		if config.Cfg.Temp.CacheTTL > 0 {
			common.PurgeFileAfter(common.GetDownloadedFilePath(task.FileName),
				time.Duration(config.Cfg.Temp.CacheTTL)*time.Second)
		} else {
			if err := common.PurgeFile(common.GetDownloadedFilePath(task.FileName)); err != nil {
				logger.L.Errorf("Failed to purge file: %s", err)
			}
		}
	}()
	if task.StoragePath == "" {
		task.StoragePath = task.FileName
	}

	if err := storage.Save(task.Storage, task.Ctx, common.GetDownloadedFilePath(task.FileName), task.StoragePath); err != nil {
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
			bot.Bot.EditMessageText(&telego.EditMessageTextParams{
				ChatID:    telegoutil.ID(task.ChatID),
				MessageID: task.ReplyMessageID,
				Text:      "文件转存完成",
			})
			if err := dao.DeleteReceivedFileByFileID(task.FileID); err != nil {
				logger.L.Errorf("Failed to delete received file: %s", err)
			}
		case types.Failed:
			logger.L.Errorf("Task failed: %s", task.String())
			bot.Bot.EditMessageText(&telego.EditMessageTextParams{
				ChatID:    telegoutil.ID(task.ChatID),
				MessageID: task.ReplyMessageID,
				Text:      "文件转存失败:" + "\n" + task.Error.Error(),
			})
			if err := dao.DeleteReceivedFileByFileID(task.FileID); err != nil {
				logger.L.Errorf("Failed to delete received file: %s", err)
			}
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
