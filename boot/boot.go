package boot

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/log"

	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/settings"
)

type Result struct {
	Config   config.Config
	Settings *settings.Service
	DataDir  string
	DBPath   string
	Listen   string
	NoWeb    bool
}

func Init(ctx context.Context, opts Options) (Result, error) {
	if err := os.MkdirAll(opts.DataDir, 0o755); err != nil {
		return Result{}, err
	}
	database.Init(ctx, opts.DBPath)
	if !settings.LegacyImported(ctx) {
		if err := settings.ImportLegacy(ctx, opts.ConfigPath, opts.DataDir); err != nil {
			log.Errorf("Failed to import legacy configuration: %v", err)
		}
	}
	svc, err := settings.NewService(ctx, opts.DataDir)
	if err != nil {
		return Result{}, err
	}
	if opts.Listen != "" {
		svc.SetListenOverride(opts.Listen)
	}
	cfg := svc.Config()
	if opts.InitWeb && !opts.NoWeb && cfg.Web.Enable {
		// The generated password is emitted at Warn; force a level that shows it.
		log.SetLevel(log.InfoLevel)
		if err := svc.EnsureAdmin(os.Getenv("SAVEANY_ADMIN_PASSWORD")); err != nil {
			return Result{}, err
		}
	}
	if level, err := log.ParseLevel(strings.TrimSpace(cfg.Log.Level)); err == nil {
		log.SetLevel(level)
	} else {
		log.Warn("Invalid log level, using info", "level", cfg.Log.Level)
	}
	i18n.Init(cfg.Lang)
	cache.Init()
	if cfg.Proxy != "" {
		transport, err := config.ProxyTransport(cfg.Proxy)
		if err != nil {
			log.Errorf("Invalid proxy configuration: %v", err)
		} else {
			http.DefaultTransport = transport
		}
	}
	return Result{
		Config:   cfg,
		Settings: svc,
		DataDir:  opts.DataDir,
		DBPath:   opts.DBPath,
		Listen:   cfg.Web.Listen,
		NoWeb:    opts.NoWeb,
	}, nil
}
