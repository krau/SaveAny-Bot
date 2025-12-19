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

	exitChan, err := initAll(ctx, cmd)
	if err != nil {
		logger.Fatal("Init failed", "error", err)
	}
	go func() {
		<-exitChan
		cancel()
	}()

	core.Run(ctx)

	<-ctx.Done()
	logger.Info("Exiting...")
	defer logger.Info("Exit complete")
	cleanCache()
}

func initAll(ctx context.Context, cmd *cobra.Command) (<-chan struct{}, error) {
	configFile := config.GetConfigFile(cmd)
	if err := config.Init(ctx, configFile); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	cache.Init()
	logger := log.FromContext(ctx)
	i18n.Init(config.C().Lang)
	logger.Info("Initializing...")
	database.Init(ctx)
	storage.LoadStorages(ctx)
	if config.C().Parser.PluginEnable {
		for _, dir := range config.C().Parser.PluginDirs {
			if err := parsers.LoadPlugins(ctx, dir); err != nil {
				logger.Error("Failed to load parser plugins", "dir", dir, "error", err)
			} else {
				logger.Debug("Loaded parser plugins from directory", "dir", dir)
			}
		}
	}
	if config.C().Telegram.Userbot.Enable {
		_, err := userclient.Login(ctx)
		if err != nil {
			logger.Fatal("User login failed", "error", err)
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
			log.Error("Invalid cache directory", "path", config.C().Temp.BasePath)
			return
		}
		currentDir, err := os.Getwd()
		if err != nil {
			log.Error("Failed to get working directory", "error", err)
			return
		}
		cachePath := filepath.Join(currentDir, config.C().Temp.BasePath)
		cachePath, err = filepath.Abs(cachePath)
		if err != nil {
			log.Error("Failed to get absolute cache path", "error", err)
			return
		}
		log.Info("Cleaning cache directory", "path", cachePath)
		if err := fsutil.RemoveAllInDir(cachePath); err != nil {
			log.Error("Failed to clean cache directory", "error", err)
		}
	}
}
