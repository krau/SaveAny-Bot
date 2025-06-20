package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"slices"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot"
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/spf13/cobra"
)

func Run(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	logger := log.NewWithOptions(os.Stdout, log.Options{
		Level:           log.DebugLevel,
		ReportTimestamp: true,
		TimeFormat:      time.TimeOnly,
		ReportCaller:    true,
	})
	ctx = log.WithContext(ctx, logger)

	initAll(ctx)
	core.Run(ctx)

	<-ctx.Done()
	logger.Info(i18n.T(i18nk.Exiting))
	defer logger.Info(i18n.T(i18nk.Bye))
	cleanCache()
}

func initAll(ctx context.Context) {
	if err := config.Init(ctx); err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(1)
	}
	cache.Init()
	logger := log.FromContext(ctx)
	i18n.Init(config.Cfg.Lang)
	logger.Info(i18n.T(i18nk.Initing))
	if config.Cfg.Telegram.Userbot.Enable {
		uc, err := userclient.Login(ctx)
		if err != nil {
			logger.Fatalf("User client login failed: %s", err)
		}
		logger.Infof("User client logged in as %s", uc.Self.FirstName)
	}
	database.Init(ctx)
	storage.LoadStorages(ctx)

	bot.Init(ctx)
}

func cleanCache() {
	if config.Cfg.NoCleanCache {
		return
	}
	if config.Cfg.Temp.BasePath != "" && !config.Cfg.Stream {
		if slices.Contains([]string{"/", ".", "\\", ".."}, filepath.Clean(config.Cfg.Temp.BasePath)) {
			log.Error(i18n.T(i18nk.InvalidCacheDir, map[string]any{
				"Path": config.Cfg.Temp.BasePath,
			}))
			return
		}
		currentDir, err := os.Getwd()
		if err != nil {
			log.Error(i18n.T(i18nk.GetWorkdirFailed, map[string]any{
				"Error": err,
			}))
			return
		}
		cachePath := filepath.Join(currentDir, config.Cfg.Temp.BasePath)
		cachePath, err = filepath.Abs(cachePath)
		if err != nil {
			log.Error(i18n.T(i18nk.GetCacheAbsPathFailed, map[string]any{
				"Error": err,
			}))
			return
		}
		log.Info(i18n.T(i18nk.CleaningCache, map[string]any{
			"Path": cachePath,
		}))
		if err := fsutil.RemoveAllInDir(cachePath); err != nil {
			log.Error(i18n.T(i18nk.CleanCacheFailed, map[string]any{
				"Error": err,
			}))
		}
	}
}
