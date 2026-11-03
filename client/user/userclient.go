package user

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	gotgproto "github.com/krau/mygotg"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/dispatcher/handlers"
	"github.com/krau/mygotg/dispatcher/handlers/filters"
	gotgerrors "github.com/krau/mygotg/errors"
	"github.com/krau/mygotg/ext"
	"github.com/krau/mygotg/session"

	"github.com/krau/SaveAny-Bot/client/middleware"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
)

var ErrLoginRequired = errors.New("telegram login required")

type Status struct {
	State    string
	ID       int64
	Username string
	Phone    string
	Error    string
}

type failConversator struct{}

func (failConversator) AskPhoneNumber() (string, error) { return "", ErrLoginRequired }
func (failConversator) AskCode() (string, error)        { return "", ErrLoginRequired }
func (failConversator) AskPassword() (string, error)    { return "", ErrLoginRequired }
func (failConversator) AuthStatus(gotgproto.AuthStatus) {}

var (
	ucMu        sync.RWMutex
	uc          *gotgproto.Client
	uctx        *ext.Context
	ustatus     = Status{State: "stopped"}
	starting    bool
	generation  uint64
	startCancel context.CancelFunc
)

func CurrentStatus() Status {
	ucMu.RLock()
	defer ucMu.RUnlock()
	return ustatus
}

// GetCtx returns the userbot context, or nil when the userbot is not running.
func GetCtx() *ext.Context {
	ucMu.RLock()
	defer ucMu.RUnlock()
	return uctx
}

// Start connects the userbot, asking conv for credentials when the session is
// not authorized. Passing nil uses a conversator that fails with
// ErrLoginRequired.
func Start(ctx context.Context, conv gotgproto.AuthConversator, phone string) error {
	ucMu.Lock()
	if uc != nil {
		ucMu.Unlock()
		return nil
	}
	if starting {
		ucMu.Unlock()
		return errors.New("userbot is already starting")
	}
	starting = true
	generation++
	gen := generation
	startCtx, cancelStart := context.WithCancel(ctx)
	startCancel = cancelStart
	ustatus = Status{State: "starting"}
	ucMu.Unlock()

	autoStart := conv == nil
	if conv == nil {
		conv = failConversator{}
	}
	logger := log.FromContext(ctx)
	logger.Debug("Logging in user client")

	type result struct {
		client *gotgproto.Client
		err    error
	}
	res := make(chan result, 1)
	go func() {
		sent := false
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("Userbot client initialization panicked: %v\n%s", r, debug.Stack())
				if !sent {
					res <- result{nil, fmt.Errorf("userbot client initialization panicked: %v", r)}
				}
			}
		}()
		resolver, err := tgutil.NewConfigProxyResolver()
		if err != nil {
			res <- result{nil, err}
			sent = true
			return
		}
		tclient, err := gotgproto.NewClient(
			config.C().Telegram.AppID,
			config.C().Telegram.AppHash,
			gotgproto.ClientTypePhone(phone),
			&gotgproto.ClientOpts{
				Session:             session.SqlSession(database.GetDialect(config.C().Telegram.Userbot.Session)),
				AuthConversator:     conv,
				NoAutoAuth:          autoStart,
				Context:             startCtx,
				DisableCopyright:    true,
				DeferUpdateRecovery: true,
				Resolver:            resolver,
				MaxRetries:          config.C().Telegram.RpcRetry,
				AutoFetchReply:      true,
				Middlewares:         middleware.NewDefaultMiddlewares(startCtx, 5*time.Minute),
				ErrorHandler: func(ctx *ext.Context, u *ext.Update, s string) error {
					log.FromContext(ctx).Errorf("Unhandled error: %s", s)
					return dispatcher.EndGroups
				},
			},
		)
		res <- result{tclient, err}
		sent = true
	}()

	var r result
	select {
	case <-startCtx.Done():
		ucMu.Lock()
		if generation == gen {
			starting = false
			generation++
			startCancel = nil
			ustatus = Status{State: "stopped"}
		}
		ucMu.Unlock()
		return startCtx.Err()
	case r = <-res:
	}
	if r.err != nil {
		state := "error"
		if errors.Is(r.err, ErrLoginRequired) || (autoStart && errors.Is(r.err, gotgerrors.ErrSessionUnauthorized)) {
			state = "login_required"
		}
		ucMu.Lock()
		if generation == gen {
			starting = false
			startCancel = nil
			ustatus = Status{State: state, Error: r.err.Error(), Phone: phone}
		}
		ucMu.Unlock()
		return r.err
	}

	if startCtx.Err() != nil {
		r.client.Stop()
		ucMu.Lock()
		if generation == gen {
			starting = false
			startCancel = nil
			ustatus = Status{State: "stopped"}
		}
		ucMu.Unlock()
		return startCtx.Err()
	}
	tclient := r.client
	tclient.Dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, func(ctx *ext.Context, u *ext.Update) error {
		switch u.UpdateClass.(type) {
		case *tg.UpdateEditChannelMessage, *tg.UpdateEditMessage, *tg.UpdateDeleteChannelMessages, *tg.UpdateDeleteMessages:
			return dispatcher.EndGroups
		}
		chatID := u.EffectiveChat().GetID()
		watchChats, err := database.GetWatchChatsByChatID(ctx, chatID)
		if err != nil || len(watchChats) == 0 {
			return dispatcher.EndGroups
		}
		return dispatcher.ContinueGroups
	}))
	tclient.Dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, handleMediaMessage))
	if err := tclient.StartUpdateRecovery(startCtx); err != nil {
		tclient.Stop()
		ucMu.Lock()
		if generation == gen {
			starting = false
			startCancel = nil
			ustatus = Status{State: "error", Error: err.Error(), Phone: phone}
		}
		ucMu.Unlock()
		return err
	}

	status := Status{State: "running", Phone: phone}
	if tclient.Self != nil {
		status.ID = tclient.Self.ID
		status.Username = tclient.Self.Username
	}
	ucMu.Lock()
	if generation != gen {
		ucMu.Unlock()
		tclient.Stop()
		return errors.New("userbot start canceled")
	}
	uc = tclient
	uctx = tclient.CreateContext()
	starting = false
	startCancel = nil
	ustatus = status
	ucMu.Unlock()

	if tclient.Self != nil {
		logger.Infof("User client logged in successfully: %s %s", tclient.Self.FirstName, tclient.Self.LastName)
	} else {
		logger.Info("User client logged in successfully")
	}
	return nil
}

func Stop() {
	ucMu.Lock()
	client := uc
	uc = nil
	uctx = nil
	starting = false
	if startCancel != nil {
		startCancel()
		startCancel = nil
	}
	generation++
	ustatus = Status{State: "stopped"}
	ucMu.Unlock()
	if client != nil {
		client.Stop()
	}
}

// Logout stops the userbot and removes its persisted session.
func Logout() error {
	Stop()
	sessionPath := config.C().Telegram.Userbot.Session
	if sessionPath == "" {
		return nil
	}
	for _, path := range []string{sessionPath, sessionPath + "-wal", sessionPath + "-shm"} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
