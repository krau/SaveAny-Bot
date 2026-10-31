package bot

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	gotgproto "github.com/krau/mygotg"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
	"github.com/krau/mygotg/session"

	"github.com/krau/SaveAny-Bot/client/bot/handlers"
	"github.com/krau/SaveAny-Bot/client/middleware"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
)

type Status struct {
	State     string
	ID        int64
	Username  string
	Error     string
	StartedAt time.Time
}

type manager struct {
	mu             sync.RWMutex
	client         *gotgproto.Client
	ectx           *ext.Context
	status         Status
	starting       bool
	generation     uint64
	startCancel    context.CancelFunc
	restartHandler func()
}

var mgr = &manager{status: Status{State: "stopped"}}

func SetRestartHandler(fn func()) {
	mgr.mu.Lock()
	mgr.restartHandler = fn
	mgr.mu.Unlock()
}

func CurrentStatus() Status {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.status
}

// ExtContext returns the bot context, or nil when the bot is not running.
func ExtContext() *ext.Context {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.ectx
}

// Start connects the bot using the current configuration. It is idempotent
// while a client is already running.
func Start(ctx context.Context) error {
	mgr.mu.Lock()
	if mgr.client != nil {
		mgr.mu.Unlock()
		return nil
	}
	if mgr.starting {
		mgr.mu.Unlock()
		return errors.New("bot is already starting")
	}
	mgr.starting = true
	mgr.generation++
	gen := mgr.generation
	startCtx, cancelStart := context.WithCancel(ctx)
	mgr.startCancel = cancelStart
	mgr.status = Status{State: "starting"}
	mgr.mu.Unlock()

	logger := log.FromContext(ctx)
	logger.Info("Initializing Bot...")

	type result struct {
		client *gotgproto.Client
		err    error
	}
	resultChan := make(chan result, 1)

	go func() {
		resolver, err := tgutil.NewConfigProxyResolver()
		if err != nil {
			resultChan <- result{nil, err}
			return
		}
		client, err := gotgproto.NewClient(
			config.C().Telegram.AppID,
			config.C().Telegram.AppHash,
			gotgproto.ClientTypeBot(config.C().Telegram.Token),
			&gotgproto.ClientOpts{
				Session:             session.SqlSession(database.GetDialect(config.C().DB.Session)),
				DisableCopyright:    true,
				DeferUpdateRecovery: true,
				Middlewares:         middleware.NewDefaultMiddlewares(startCtx, 5*time.Minute),
				Resolver:            resolver,
				Context:             startCtx,
				MaxRetries:          config.C().Telegram.RpcRetry,
				AutoFetchReply:      true,
				ErrorHandler: func(ctx *ext.Context, u *ext.Update, s string) error {
					if s == "SAVEANTBOT-RESTART" {
						mgr.mu.RLock()
						handler := mgr.restartHandler
						mgr.mu.RUnlock()
						if handler != nil {
							handler()
						}
						return dispatcher.EndGroups
					}
					log.FromContext(ctx).Errorf("unhandled error: %s", s)
					return dispatcher.EndGroups
				},
			},
		)
		resultChan <- result{client, err}
	}()

	var res result
	select {
	case <-startCtx.Done():
		mgr.mu.Lock()
		if mgr.generation == gen {
			mgr.starting = false
			mgr.generation++
			mgr.startCancel = nil
			mgr.status = Status{State: "stopped"}
		}
		mgr.mu.Unlock()
		return startCtx.Err()
	case res = <-resultChan:
	}

	if res.err != nil {
		mgr.mu.Lock()
		if mgr.generation == gen {
			mgr.starting = false
			mgr.startCancel = nil
			mgr.status = Status{State: "error", Error: res.err.Error()}
		}
		mgr.mu.Unlock()
		return res.err
	}

	mgr.mu.RLock()
	stale := mgr.generation != gen || startCtx.Err() != nil
	mgr.mu.RUnlock()
	if stale {
		res.client.Stop()
		return errors.New("bot start canceled")
	}

	ectx := res.client.CreateContext()
	handlers.Register(res.client.Dispatcher)
	if err := res.client.StartUpdateRecovery(startCtx); err != nil {
		res.client.Stop()
		mgr.mu.Lock()
		if mgr.generation == gen {
			mgr.starting = false
			mgr.startCancel = nil
			mgr.status = Status{State: "error", Error: err.Error()}
		}
		mgr.mu.Unlock()
		return err
	}

	commands := make([]tg.BotCommand, 0, len(handlers.CommandHandlers))
	for _, info := range handlers.CommandHandlers {
		commands = append(commands, tg.BotCommand{Command: info.Cmd, Description: i18n.T(info.Desc)})
	}
	if _, err := res.client.API().BotsSetBotCommands(startCtx, &tg.BotsSetBotCommandsRequest{
		Scope:    &tg.BotCommandScopeDefault{},
		Commands: commands,
	}); err != nil {
		logger.Errorf("Failed to set bot commands: %v", err)
	}

	status := Status{State: "running", StartedAt: time.Now()}
	if res.client.Self != nil {
		status.ID = res.client.Self.ID
		status.Username = res.client.Self.Username
	}

	mgr.mu.Lock()
	if mgr.generation != gen {
		mgr.mu.Unlock()
		res.client.Stop()
		return errors.New("bot start canceled")
	}
	mgr.client = res.client
	mgr.ectx = ectx
	mgr.starting = false
	mgr.startCancel = nil
	mgr.status = status
	mgr.mu.Unlock()

	logger.Info("Bot initialization completed.")
	return nil
}

func Stop() {
	mgr.mu.Lock()
	client := mgr.client
	mgr.client = nil
	mgr.ectx = nil
	mgr.starting = false
	if mgr.startCancel != nil {
		mgr.startCancel()
		mgr.startCancel = nil
	}
	mgr.generation++
	mgr.status = Status{State: "stopped"}
	mgr.mu.Unlock()
	if client != nil {
		client.Stop()
	}
}
