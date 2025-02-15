package core

import (
	"fmt"
	"os"
	"time"

	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func saveFileWithRetry(task *types.Task, localFilePath string) error {
	for i := 0; i <= config.Cfg.Retry; i++ {
		if err := storage.Save(task.Storage, task.Ctx, localFilePath, task.StoragePath); err != nil {
			if i == config.Cfg.Retry {
				return fmt.Errorf("failed to save file: %w", err)
			}
			logger.L.Errorf("Failed to save file: %s, retrying...", err)
			continue
		}
		return nil
	}
	return nil
}

func processPhoto(task *types.Task, cachePath string) error {
	res, err := bot.Client.API().UploadGetFile(task.Ctx, &tg.UploadGetFileRequest{
		Location: task.File.Location,
		Offset:   0,
		Limit:    1024 * 1024,
	})
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	result, ok := res.(*tg.UploadFile)
	if !ok {
		return fmt.Errorf("unexpected type %T", res)
	}

	if err := os.WriteFile(cachePath, result.Bytes, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	defer cleanCacheFile(cachePath)

	logger.L.Infof("Downloaded file: %s", cachePath)

	return saveFileWithRetry(task, cachePath)
}

func getProgressBar(progress float64, totalCount int) string {
	bar := ""
	barSize := 100 / totalCount
	for i := 0; i < totalCount; i++ {
		if int(progress)/barSize > i {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return bar
}

func cleanCacheFile(destPath string) {
	if config.Cfg.Temp.CacheTTL > 0 {
		common.RmFileAfter(destPath, time.Duration(config.Cfg.Temp.CacheTTL)*time.Second)
	} else {
		if err := os.Remove(destPath); err != nil {
			logger.L.Errorf("Failed to purge file: %s", err)
		}
	}
}

func calculateBarTotalCount(fileSize int64) int {
	barTotalCount := 5
	if fileSize > 1024*1024*1000 {
		barTotalCount = 40
	} else if fileSize > 1024*1024*500 {
		barTotalCount = 20
	} else if fileSize > 1024*1024*200 {
		barTotalCount = 10
	}
	return barTotalCount
}

func getSpeed(bytesRead int64, startTime time.Time) string {
	if startTime.IsZero() {
		return "0MB/s"
	}
	elapsed := time.Since(startTime)
	speed := float64(bytesRead) / 1024 / 1024 / elapsed.Seconds()
	return fmt.Sprintf("%.2fMB/s", speed)
}
