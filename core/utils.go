package core

import (
	"fmt"
	"os"
	"time"

	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func saveFileWithRetry(task *types.Task, destPath string) error {
	for i := 0; i <= config.Cfg.Retry; i++ {
		if err := storage.Save(task.Storage, task.Ctx, destPath, task.StoragePath); err != nil {
			if i == config.Cfg.Retry {
				return fmt.Errorf("Failed to save file: %w", err)
			}
			logger.L.Errorf("Failed to save file: %s, retrying...", err)
			continue
		}
		return nil
	}
	return nil
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
		barTotalCount = 50
	} else if fileSize > 1024*1024*500 {
		barTotalCount = 20
	} else if fileSize > 1024*1024*200 {
		barTotalCount = 10
	}
	return barTotalCount
}
