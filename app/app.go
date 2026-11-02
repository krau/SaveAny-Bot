package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/mygotg/ext"

	"github.com/krau/SaveAny-Bot/api"
	"github.com/krau/SaveAny-Bot/client/bot"
	"github.com/krau/SaveAny-Bot/client/bot/handlers"
	"github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/client/worker"
	"github.com/krau/SaveAny-Bot/common/selfupdate"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/resume"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/parsers"
	"github.com/krau/SaveAny-Bot/settings"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/web"
)

type Options struct {
	DataDir string
	DBPath  string
	Listen  string
	NoWeb   bool
}

type App struct {
	ctx    context.Context
	cancel context.CancelFunc
	svc    *settings.Service
	opts   Options
	web    *web.Server

	applyMu               sync.Mutex
	mu                    sync.Mutex
	resumeDone            chan struct{}
	appliedBotCred        string
	appliedUserbotSession string
	flowMu                sync.Mutex
	flow                  *loginFlow
}

func New(parent context.Context, svc *settings.Service, opts Options) *App {
	ctx, cancel := context.WithCancel(parent)
	return &App{ctx: ctx, cancel: cancel, svc: svc, opts: opts}
}

func (a *App) Done() <-chan struct{} { return a.ctx.Done() }

func (a *App) Start() error {
	bot.SetRestartHandler(a.RequestRestart)
	if err := database.MarkInterruptedTaskRecords(a.ctx); err != nil {
		log.Errorf("Failed to mark interrupted tasks: %v", err)
	}
	worker.Init(a.ctx, a.opts.DataDir)
	core.Run(a.ctx)

	cfg := a.svc.Config()
	if cfg.Parser.PluginEnable {
		for _, dir := range cfg.Parser.PluginDirs {
			if err := parsers.LoadPlugins(a.ctx, dir); err != nil {
				log.Error("Failed to load parser plugins", "dir", dir, "error", err)
			}
		}
	}
	go handlers.ListenUserbotMediaEvents(a.ctx, func() *ext.Context {
		if botCtx := bot.ExtContext(); botCtx != nil {
			return botCtx
		}
		return user.GetCtx()
	})

	if !a.opts.NoWeb && cfg.Web.Enable {
		webToken := strings.TrimSpace(cfg.Web.APIToken)
		if webToken == "" {
			webToken = config.GenerateToken()
		}
		a.web = web.New(a.opts.Listen, webToken, api.NewHandler(api.Deps{
			Ctx:      a.ctx,
			Settings: a.svc,
			Telegram: a,
			DataDir:  a.opts.DataDir,
			DBPath:   a.opts.DBPath,
			WebToken: webToken,
		}))
		if err := a.web.Start(a.ctx); err != nil {
			return err
		}
	}

	go func() {
		if err := a.ApplyTelegram(a.ctx); err != nil {
			log.Errorf("Failed to apply telegram settings: %v", err)
		}
	}()
	a.resumeDone = make(chan struct{})
	go func() {
		defer close(a.resumeDone)
		resume.Run(a.ctx)
	}()
	return nil
}

func (a *App) ApplyTelegram(ctx context.Context) error {
	a.applyMu.Lock()
	defer a.applyMu.Unlock()
	cfg := a.svc.Config()
	cred := strings.Join([]string{
		cfg.Telegram.Token,
		fmt.Sprintf("%d", cfg.Telegram.AppID),
		fmt.Sprintf("%d", cfg.Telegram.RpcRetry),
		cfg.Telegram.AppHash,
		fmt.Sprintf("%t", cfg.Telegram.Proxy.Enable),
		cfg.Telegram.Proxy.URL,
	}, "|")

	userbotSession := cfg.Telegram.Userbot.Session
	a.mu.Lock()
	changed := cred != a.appliedBotCred
	userbotSessionChanged := userbotSession != a.appliedUserbotSession
	a.appliedBotCred = cred
	a.appliedUserbotSession = userbotSession
	a.mu.Unlock()

	botStatus := bot.CurrentStatus()
	switch {
	case cfg.Telegram.Token == "":
		if botStatus.State != "stopped" {
			bot.Stop()
		}
	case changed:
		bot.Stop()
		go func() {
			if err := bot.Start(a.ctx); err != nil {
				log.Errorf("Failed to start bot: %v", err)
			}
		}()
	case botStatus.State == "stopped" || botStatus.State == "error":
		go func() {
			if err := bot.Start(a.ctx); err != nil {
				log.Errorf("Failed to start bot: %v", err)
			}
		}()
	}

	userStatus := user.CurrentStatus()
	if userbotSessionChanged && (userStatus.State == "running" || userStatus.State == "starting") {
		user.Stop()
		userStatus = user.CurrentStatus()
	}
	if !cfg.Telegram.Userbot.Enable {
		if userStatus.State == "running" || userStatus.State == "starting" {
			user.Stop()
		}
	} else if userStatus.State == "stopped" || userStatus.State == "error" || userStatus.State == "login_required" {
		go func() {
			if err := user.Start(a.ctx, nil, cfg.Telegram.Userbot.Phone); err != nil && !errors.Is(err, user.ErrLoginRequired) {
				log.Errorf("Failed to start userbot: %v", err)
			}
		}()
	}
	return nil
}

func (a *App) RequestRestart() {
	selfupdate.RequestRestart()
	a.cancel()
}

func (a *App) Stop() {
	a.cancel()
	if a.resumeDone != nil {
		select {
		case <-a.resumeDone:
		case <-time.After(5 * time.Second):
			log.Warn("Timed out waiting for queued task recovery to stop")
		}
	}
	if a.web != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := a.web.Stop(stopCtx); err != nil {
			log.Warnf("Failed to stop web server: %v", err)
		}
		cancel()
	}
	bot.Stop()
	user.Stop()
	worker.StopAll()
	core.Close()
	if !core.WaitWorkers(10 * time.Second) {
		log.Warn("Timed out waiting for task workers to stop")
	}
	storage.CloseAll()
	if err := database.Close(); err != nil {
		log.Warnf("Failed to close database: %v", err)
	}
}
