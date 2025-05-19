package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"slices"

	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/spf13/cobra"
)

func Run(_ *cobra.Command, _ []string) {
	InitAll()
	core.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	common.Log.Info(sig, ", exitting...")
	defer common.Log.Info("Bye!")
	if config.Cfg.NoCleanCache {
		return
	}
	if config.Cfg.Temp.BasePath != "" && !config.Cfg.Stream {
		if slices.Contains([]string{"/", ".", "\\", ".."}, filepath.Clean(config.Cfg.Temp.BasePath)) {
			common.Log.Error("无效的缓存文件夹: ", config.Cfg.Temp.BasePath)
			return
		}
		currentDir, err := os.Getwd()
		if err != nil {
			common.Log.Error("获取工作目录失败: ", err)
			return
		}
		cachePath := filepath.Join(currentDir, config.Cfg.Temp.BasePath)
		cachePath, err = filepath.Abs(cachePath)
		if err != nil {
			common.Log.Error("获取缓存绝对路径失败: ", err)
			return
		}
		common.Log.Info("正在清理缓存文件夹: ", cachePath)
		if err := common.RemoveAllInDir(cachePath); err != nil {
			common.Log.Error("清理缓存失败: ", err)
		}
	}
}

func InitAll() {
	if err := config.Init(); err != nil {
		fmt.Println("加载配置文件失败: ", err)
		os.Exit(1)
	}
	common.InitLogger()
	common.Log.Info("正在启动 SaveAny-Bot...")
	dao.Init()
	storage.LoadStorages()
	common.Init()
	bot.Init()
}
