package cmd

import (
	"context"
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
	"github.com/krau/SaveAny-Bot/i18n"
	"github.com/krau/SaveAny-Bot/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/userclient"
	"github.com/spf13/cobra"
)

func Run(_ *cobra.Command, _ []string) {
	InitAll()
	core.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	common.Log.Info(sig, i18n.T(i18nk.Exiting))
	defer common.Log.Info(i18n.T(i18nk.Bye))
	if config.Cfg.NoCleanCache {
		return
	}
	if config.Cfg.Temp.BasePath != "" && !config.Cfg.Stream {
		if slices.Contains([]string{"/", ".", "\\", ".."}, filepath.Clean(config.Cfg.Temp.BasePath)) {
			common.Log.Error(i18n.T(i18nk.InvalidCacheDir, map[string]any{
				"Path": config.Cfg.Temp.BasePath,
			}))
			return
		}
		currentDir, err := os.Getwd()
		if err != nil {
			common.Log.Error(i18n.T(i18nk.GetWorkdirFailed, map[string]any{
				"Error": err,
			}))
			return
		}
		cachePath := filepath.Join(currentDir, config.Cfg.Temp.BasePath)
		cachePath, err = filepath.Abs(cachePath)
		if err != nil {
			common.Log.Error(i18n.T(i18nk.GetCacheAbsPathFailed, map[string]any{
				"Error": err,
			}))
			return
		}
		common.Log.Info(i18n.T(i18nk.CleaningCache, map[string]any{
			"Path": cachePath,
		}))
		if err := common.RemoveAllInDir(cachePath); err != nil {
			common.Log.Error(i18n.T(i18nk.CleanCacheFailed, map[string]any{
				"Error": err,
			}))
		}
	}
}

func InitAll() {
	if err := config.Init(); err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(1)
	}
	common.InitLogger()
	i18n.Init(config.Cfg.Lang)
	common.Log.Info(i18n.T(i18nk.Initing))
	dao.Init()
	storage.LoadStorages()
	common.Init()
	if config.Cfg.Telegram.Userbot.Enable {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()
		uc, err := userclient.Login(ctx)
		if err != nil {
			common.Log.Errorf("User client login failed: %s", err)
			os.Exit(1)
		}
		common.Log.Infof("User client logged in as %s", uc.Self.FirstName)
	}
	bot.Init()
}
