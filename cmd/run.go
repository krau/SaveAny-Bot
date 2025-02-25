package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/spf13/cobra"
)

func Run(_ *cobra.Command, _ []string) {
	InitAll()
	core.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.L.Info(sig, ", exitting...")
	defer logger.L.Info("Bye!")
	if config.Cfg.NoCleanCache {
		return
	}
	if config.Cfg.Temp.BasePath != "" {
		for _, path := range []string{"/", ".", "\\", ".."} {
			if filepath.Clean(config.Cfg.Temp.BasePath) == path {
				logger.L.Error("Invalid cache dir: ", config.Cfg.Temp.BasePath)
				return
			}
		}
		currentDir, err := os.Getwd()
		if err != nil {
			logger.L.Error("Failed to get current dir: ", err)
			return
		}
		cachePath := filepath.Join(currentDir, config.Cfg.Temp.BasePath)
		cachePath, err = filepath.Abs(cachePath)
		if err != nil {
			logger.L.Error("Failed to get absolute path: ", err)
			return
		}
		logger.L.Info("Cleaning cache dir: ", cachePath)
		if err := os.RemoveAll(cachePath); err != nil {
			logger.L.Error("Failed to clean cache dir: ", err)
		}
	}
}

func InitAll() {
	if err := config.Init(); err != nil {
		fmt.Println("加载配置文件失败: ", err)
		os.Exit(1)
	}
	logger.InitLogger()
	logger.L.Info("正在启动 SaveAny-Bot...")
	dao.Init()
	storage.LoadStorages()
	common.Init()
	bot.Init()
}
