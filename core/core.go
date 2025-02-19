package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/gabriel-vasile/mimetype"

	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func processPendingTask(task *types.Task) error {
	logger.L.Debugf("Start processing task: %s", task.String())
	if task.FileName() == "" {
		task.File.FileName = fmt.Sprintf("%d_%d_%s", task.FileChatID, task.FileMessageID, task.File.Hash())
	}
	cacheDestPath := filepath.Join(config.Cfg.Temp.BasePath, task.FileName())
	cacheDestPath, err := filepath.Abs(cacheDestPath)
	if err != nil {
		return fmt.Errorf("处理路径失败: %w", err)
	}
	if err := fileutil.CreateDir(filepath.Dir(cacheDestPath)); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	if task.StoragePath == "" {
		task.StoragePath = task.File.FileName
	}

	taskStorage, err := storage.GetStorageByUserIDAndName(task.UserID, task.StorageName)
	if err != nil {
		return err
	}
	task.StoragePath = taskStorage.JoinStoragePath(*task)

	if task.File.FileSize == 0 {
		return processPhoto(task, taskStorage, cacheDestPath)
	}

	ctx := task.Ctx.(*ext.Context)

	barTotalCount := calculateBarTotalCount(task.File.FileSize)

	progressCallback := func(bytesRead, contentLength int64) {
		progress := float64(bytesRead) / float64(contentLength) * 100
		logger.L.Tracef("Downloading %s: %.2f%%", task.String(), progress)
		if task.File.FileSize < 1024*1024*50 || int(progress)%(100/barTotalCount) != 0 {
			return
		}
		text, entities := buildProgressMessageEntity(task, barTotalCount, bytesRead, task.StartTime, progress)
		ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
			Message:  text,
			Entities: entities,
			ID:       task.ReplyMessageID,
		})
	}

	text, entities := buildProgressMessageEntity(task, barTotalCount, 0, task.StartTime, 0)
	ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
		Message:  text,
		Entities: entities,
		ID:       task.ReplyMessageID,
	})

	readCloser, err := NewTelegramReader(task.Ctx, bot.Client, &task.File.Location,
		0, task.File.FileSize-1, task.File.FileSize,
		progressCallback, task.File.FileSize/100)
	if err != nil {
		return fmt.Errorf("创建下载失败: %w", err)
	}
	defer readCloser.Close()

	dest, err := os.Create(cacheDestPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer dest.Close()
	task.StartTime = time.Now()
	if _, err := io.CopyN(dest, readCloser, task.File.FileSize); err != nil {
		return fmt.Errorf("下载文件失败: %w", err)
	}
	defer cleanCacheFile(cacheDestPath)
	if path.Ext(task.FileName()) == "" {
		mimeType, err := mimetype.DetectFile(cacheDestPath)
		if err != nil {
			logger.L.Errorf("Failed to detect mime type: %s", err)
		} else {
			task.File.FileName = fmt.Sprintf("%s%s", task.FileName(), mimeType.Extension())
			task.StoragePath = fmt.Sprintf("%s%s", task.StoragePath, mimeType.Extension())
		}
	}

	logger.L.Infof("Downloaded file: %s", cacheDestPath)
	ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("下载完成: %s\n正在转存文件...", task.FileName()),
		ID:      task.ReplyMessageID,
	})

	return saveFileWithRetry(task, taskStorage, cacheDestPath)
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
			task.Ctx.(*ext.Context).EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
				Message: fmt.Sprintf("文件保存成功\n [%s]: %s", task.StorageName, task.StoragePath),
				ID:      task.ReplyMessageID,
			})
		case types.Failed:
			logger.L.Errorf("Task failed: %s", task.String())
			task.Ctx.(*ext.Context).EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
				Message: "文件保存失败\n" + task.Error.Error(),
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
