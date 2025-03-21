package core

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
	"golang.org/x/sync/errgroup"
)

func processPendingTask(task *types.Task) error {
	common.Log.Debugf("Start processing task: %s", task.String())
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

	ctx, ok := task.Ctx.(*ext.Context)
	if !ok {
		return fmt.Errorf("context is not *ext.Context: %T", task.Ctx)
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	task.Cancel = cancel

	if task.File.FileSize == 0 {
		return processPhoto(task, taskStorage)
	}

	downloadBuilder := Downloader.Download(bot.Client.API(), task.File.Location).WithThreads(getTaskThreads(task.File.FileSize))

	if config.Cfg.Stream {

		text, entities := buildProgressMessageEntity(task, 0, task.StartTime, 0)
		ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
			Message:     text,
			Entities:    entities,
			ID:          task.ReplyMessageID,
			ReplyMarkup: getCancelTaskMarkup(task),
		})

		pr, pw := io.Pipe()
		defer pr.Close()

		task.StartTime = time.Now()
		progressCallback := buildProgressCallback(ctx, task, getProgressUpdateCount(task.File.FileSize))

		progressStream := NewProgressStream(pw, task.File.FileSize, progressCallback)

		eg, uploadCtx := errgroup.WithContext(cancelCtx)

		eg.Go(func() error {
			return taskStorage.Save(uploadCtx, pr, task.StoragePath)
		})
		eg.Go(func() error {
			_, err := downloadBuilder.Stream(uploadCtx, progressStream)
			if closeErr := pw.CloseWithError(err); closeErr != nil {
				common.Log.Errorf("Failed to close pipe writer: %v", closeErr)
			}
			return err
		})
		if err := eg.Wait(); err != nil {
			return err
		}

		return nil
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
	_, err = downloadBuilder.Parallel(cancelCtx, dest)
	if err != nil {
		return fmt.Errorf("下载文件失败: %w", err)
	}
	defer cleanCacheFile(cacheDestPath)

	fixTaskFileExt(task, cacheDestPath)

	common.Log.Infof("Downloaded file: %s", cacheDestPath)
	ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("下载完成: %s\n正在转存文件...", task.FileName()),
		ID:      task.ReplyMessageID,
	})

	return saveFileWithRetry(cancelCtx, task, taskStorage, cacheDestPath)
}
