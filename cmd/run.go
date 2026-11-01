package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/krau/SaveAny-Bot/app"
	"github.com/krau/SaveAny-Bot/boot"
	"github.com/krau/SaveAny-Bot/common/selfupdate"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/config"
)

func Run(cmd *cobra.Command, _ []string) {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	logger := log.NewWithOptions(os.Stdout, log.Options{
		Level:           log.InfoLevel,
		ReportTimestamp: true,
		TimeFormat:      time.TimeOnly,
		ReportCaller:    true,
	})
	log.SetDefault(logger)
	ctx = log.WithContext(ctx, logger)

	selfupdate.CleanLeftovers()

	opts := boot.Parse(cmd)
	opts.InitWeb = true
	res, err := boot.Init(ctx, opts)
	if err != nil {
		logger.Fatal("Init failed", "error", err)
	}
	cfg := res.Config
	if level, err := log.ParseLevel(strings.TrimSpace(cfg.Log.Level)); err == nil {
		logger.SetLevel(level)
	}

	application := app.New(ctx, res.Settings, app.Options{
		DataDir: res.DataDir,
		DBPath:  res.DBPath,
		Listen:  res.Listen,
		NoWeb:   res.NoWeb,
	})
	if err := application.Start(); err != nil {
		logger.Fatal("Init failed", "error", err)
	}

	<-application.Done()
	logger.Info("Exiting...")
	defer logger.Info("Exit complete")
	application.Stop()
	cleanCache(res.Settings.Config())
	if selfupdate.ShouldRestart() {
		logger.Info("Restarting to run the updated version...")
		if err := selfupdate.Restart(); err != nil {
			logger.Errorf("Failed to restart: %s. Please start saveany-bot again", err)
		}
	}
}

func cleanCache(cfg config.Config) {
	if cfg.NoCleanCache {
		return
	}
	if cfg.Temp.BasePath != "" && !cfg.Stream {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Error("Failed to get working directory", "error", err)
			return
		}
		cachePath := cfg.Temp.BasePath
		if !filepath.IsAbs(cachePath) {
			cachePath = filepath.Join(currentDir, cachePath)
		}
		cachePath, err = filepath.Abs(cachePath)
		if err != nil {
			log.Error("Failed to get absolute cache path", "error", err)
			return
		}
		cachePath = filepath.Clean(cachePath)
		// Never clean the working directory or one of its ancestors.
		if rel, err := filepath.Rel(cachePath, currentDir); err != nil || rel == "." || !strings.HasPrefix(rel, "..") {
			log.Error("Refusing to clean unsafe cache directory", "path", cachePath)
			return
		}
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			return
		}
		log.Info("Cleaning cache directory", "path", cachePath)
		if err := fsutil.RemoveAllInDir(cachePath); err != nil {
			log.Error("Failed to clean cache directory", "error", err)
		}
	}
}
