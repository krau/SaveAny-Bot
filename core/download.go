package core

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
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

	ctx, ok := task.Ctx.(*ext.Context)
	if !ok {
		return fmt.Errorf("context is not *ext.Context: %T", task.Ctx)
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	task.Cancel = cancel

	downloadBuider := Downloader.Download(bot.Client.API(), task.File.Location).WithThreads(getTaskThreads(task.File.FileSize))

	// TODO: show progress for stream storage
	taskStreamStorage, isStreamStorage := taskStorage.(storage.StreamStorage)
	if config.Cfg.Stream {
		if !isStreamStorage {
			logger.L.Warnf("存储 %s 不支持流式上传", taskStorage.Name())
		} else {
			entityBuilder := entity.Builder{}
			text := fmt.Sprintf("正在处理下载任务 (流式)\n文件名: %s\n保存路径: %s",
				task.FileName(),
				fmt.Sprintf("[%s]:%s", task.StorageName, task.StoragePath),
			)
			var entities []tg.MessageEntityClass
			if err := styling.Perform(&entityBuilder,
				styling.Plain("正在处理下载任务 (流式)\n文件名: "),
				styling.Code(task.FileName()),
				styling.Plain("\n保存路径: "),
				styling.Code(fmt.Sprintf("[%s]:%s", task.StorageName, task.StoragePath)),
			); err != nil {
				logger.L.Errorf("Failed to build entities: %s", err)
			} else {
				text, entities = entityBuilder.Complete()
			}
			ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
				Message:     text,
				Entities:    entities,
				ID:          task.ReplyMessageID,
				ReplyMarkup: getCancelTaskMarkup(task),
			})
			uploadStream, err := taskStreamStorage.NewUploadStream(cancelCtx, task.StoragePath)
			if err != nil {
				return fmt.Errorf("创建上传流失败: %w", err)
			}
			defer uploadStream.Close()
			_, err = downloadBuider.Stream(cancelCtx, uploadStream)
			if err != nil {
				return fmt.Errorf("下载文件失败: %w", err)
			}
			logger.L.Infof("Uploaded file: %s", task.StoragePath)
			return nil
		}
	}

	text, entities := buildProgressMessageEntity(task, 0, task.StartTime, 0)
	ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
		Message:     text,
		Entities:    entities,
		ID:          task.ReplyMessageID,
		ReplyMarkup: getCancelTaskMarkup(task),
	})

	progressCallback := buildProgressCallback(ctx, task, getProgressUpdateCount(task.File.FileSize))
	dest, err := NewTaskLocalFile(cacheDestPath, task.File.FileSize, progressCallback)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer dest.Close()
	task.StartTime = time.Now()
	_, err = downloadBuider.Parallel(cancelCtx, dest)
	if err != nil {
		return fmt.Errorf("下载文件失败: %w", err)
	}
	defer cleanCacheFile(cacheDestPath)

	fixTaskFileExt(task, cacheDestPath)

	logger.L.Infof("Downloaded file: %s", cacheDestPath)
	ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("下载完成: %s\n正在转存文件...", task.FileName()),
		ID:      task.ReplyMessageID,
	})

	return saveFileWithRetry(cancelCtx, task, taskStorage, cacheDestPath)

}
