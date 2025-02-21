package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/fileutil"
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

	barTotalCount := calculateBarTotalCount(task.File.FileSize)
	text, entities := buildProgressMessageEntity(task, barTotalCount, 0, task.StartTime, 0)
	ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
		Message:  text,
		Entities: entities,
		ID:       task.ReplyMessageID,
	})
	progressCallback := buildProgressCallback(ctx, task, barTotalCount)
	readCloser, err := NewTelegramReader(ctx, bot.Client, &task.File.Location,
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

	fixTaskFileExt(task, cacheDestPath)

	logger.L.Infof("Downloaded file: %s", cacheDestPath)
	ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("下载完成: %s\n正在转存文件...", task.FileName()),
		ID:      task.ReplyMessageID,
	})

	return saveFileWithRetry(task, taskStorage, cacheDestPath)
}
