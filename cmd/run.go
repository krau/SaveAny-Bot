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
	"github.com/krau/SaveAny-Bot/parsers"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/spf13/cobra"
)

func Run(cmd *cobra.Command, _ []string) {
	ctx, cancel := context.WithCancel(cmd.Context())
	logger := log.NewWithOptions(os.Stdout, log.Options{
		Level:           log.DebugLevel,
		ReportTimestamp: true,
		TimeFormat:      time.TimeOnly,
		ReportCaller:    true,
	})
	ctx = log.WithContext(ctx, logger)

	exitChan, err := initAll(ctx)
	if err != nil {
		logger.Fatal("Failed to initialize", "error", err)
	}
	go func() {
		<-exitChan
		cancel()
	}()

	core.Run(ctx)

	<-ctx.Done()
	logger.Info(i18n.T(i18nk.Exiting))
	defer logger.Info(i18n.T(i18nk.Bye))
	cleanCache()
}

func initAll(ctx context.Context) (<-chan struct{}, error) {
	if err := config.Init(ctx); err != nil {
		fmt.Println("Failed to load config:", err)
		return nil, err
	}
	cache.Init()
	logger := log.FromContext(ctx)
	i18n.Init(config.C().Lang)
	logger.Info(i18n.T(i18nk.Initing))
	database.Init(ctx)
	storage.LoadStorages(ctx)
	if config.C().Parser.PluginEnable {
		for _, dir := range config.C().Parser.PluginDirs {
			if err := parsers.LoadPlugins(ctx, dir); err != nil {
				logger.Error("Failed to load parser plugins", "dir", dir, "error", err)
			} else {
				logger.Debug("Loaded parser plugins", "dir", dir)
			}
		}
	}
	if config.C().Telegram.Userbot.Enable {
		_, err := userclient.Login(ctx)
		if err != nil {
			logger.Fatalf("User client login failed: %s", err)
		}
	}
	return bot.Init(ctx), nil
}

func cleanCache() {
	if config.C().NoCleanCache {
		return
	}
	if config.C().Temp.BasePath != "" && !config.C().Stream {
		if slices.Contains([]string{"/", ".", "\\", ".."}, filepath.Clean(config.C().Temp.BasePath)) {
			log.Error(i18n.T(i18nk.InvalidCacheDir, map[string]any{
				"Path": config.C().Temp.BasePath,
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
		cachePath := filepath.Join(currentDir, config.C().Temp.BasePath)
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
