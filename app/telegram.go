package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/rs/xid"

	"github.com/krau/SaveAny-Bot/api"
	"github.com/krau/SaveAny-Bot/client/bot"
	"github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/settings"
)

func (a *App) Status(ctx context.Context) api.TelegramStatus {
	botStatus := bot.CurrentStatus()
	userStatus := user.CurrentStatus()
	cfg := a.svc.Config()

	status := api.TelegramStatus{
		Bot: api.BotStatus{
			State:     botStatus.State,
			ID:        botStatus.ID,
			Username:  botStatus.Username,
			Error:     botStatus.Error,
			StartedAt: botStatus.StartedAt,
		},
		Userbot: api.UserbotStatus{
			Enabled:  cfg.Telegram.Userbot.Enable,
			State:    userStatus.State,
			ID:       userStatus.ID,
			Username: userStatus.Username,
			Phone:    userStatus.Phone,
			Error:    userStatus.Error,
		},
	}
	a.flowMu.Lock()
	flow := a.flow
	a.flowMu.Unlock()
	if flow != nil {
		state, errMsg := flow.status()
		status.Userbot.Login = &api.LoginFlow{FlowID: flow.id, State: state, Error: errMsg}
	}
	return status
}

func (a *App) StartBot(ctx context.Context) error {
	if strings.TrimSpace(a.svc.Config().Telegram.Token) == "" {
		return errors.New("bot token is not configured")
	}
	if bot.CurrentStatus().State == "running" || bot.CurrentStatus().State == "starting" {
		return nil
	}
	go func() {
		if err := bot.Start(a.ctx); err != nil {
			log.Errorf("Failed to start bot: %v", err)
		}
	}()
	return nil
}

func (a *App) StopBot(ctx context.Context, force bool) error {
	if !force && core.ActiveTaskCount() > 0 {
		return errors.New("tasks are running")
	}
	bot.Stop()
	return nil
}

func (a *App) RestartBot(ctx context.Context, force bool) error {
	if !force && core.ActiveTaskCount() > 0 {
		return errors.New("tasks are running")
	}
	bot.Stop()
	go func() {
		if err := bot.Start(a.ctx); err != nil {
			log.Errorf("Failed to restart bot: %v", err)
		}
	}()
	return nil
}

func (a *App) StartUserbotLogin(ctx context.Context, phone string) (string, error) {
	if phone == "" {
		return "", errors.New("phone number is required")
	}
	if user.CurrentStatus().State == "running" || user.CurrentStatus().State == "starting" {
		return "", errors.New("userbot is already running")
	}
	cfg := a.svc.Config()
	if cfg.Telegram.AppID == 0 || cfg.Telegram.AppHash == "" {
		return "", errors.New("telegram app_id and app_hash are required")
	}

	a.flowMu.Lock()
	if a.flow != nil {
		state, _ := a.flow.status()
		if state != "failed" && state != "cancelled" && state != "done" {
			a.flowMu.Unlock()
			return "", errors.New("a login flow is already active")
		}
	}
	flow := newLoginFlow(a.ctx, xid.New().String(), phone)
	a.flow = flow
	a.flowMu.Unlock()

	go func() {
		err := user.Start(a.ctx, &webConversator{flow: flow}, phone)
		if err != nil {
			flow.markFailed(err)
			return
		}
		if flow.ctx.Err() != nil {
			user.Stop()
			return
		}
		patch, err := json.Marshal(map[string]any{
			"userbot": map[string]any{"enable": true, "phone": phone},
		})
		if err != nil {
			flow.markFailed(err)
			user.Stop()
			return
		}
		if _, err := a.svc.UpdateGroup(a.ctx, settings.GroupTelegram, patch); err != nil {
			flow.markFailed(err)
			user.Stop()
			return
		}
		if !flow.markDone() {
			user.Stop()
			disable, err := json.Marshal(map[string]any{
				"userbot": map[string]any{"enable": false},
			})
			if err != nil {
				log.Warnf("Failed to encode userbot disable patch: %v", err)
				return
			}
			if _, err := a.svc.UpdateGroup(a.ctx, settings.GroupTelegram, disable); err != nil {
				log.Warnf("Failed to disable userbot after login flow ended: %v", err)
			}
			return
		}
		flow.cancel()
	}()
	return flow.id, nil
}

func (a *App) SubmitUserbotCode(ctx context.Context, flowID, code string) error {
	flow, err := a.getFlow(flowID)
	if err != nil {
		return err
	}
	return flow.submitCode(code)
}

func (a *App) SubmitUserbotPassword(ctx context.Context, flowID, password string) error {
	flow, err := a.getFlow(flowID)
	if err != nil {
		return err
	}
	return flow.submitPassword(password)
}

func (a *App) CancelUserbotLogin(ctx context.Context, flowID string) error {
	flow, err := a.getFlow(flowID)
	if err != nil {
		return err
	}
	if state, _ := flow.status(); state == "done" {
		return errors.New("login already completed")
	}
	flow.cancel()
	flow.setState("cancelled", "")
	user.Stop()
	return nil
}

func (a *App) LogoutUserbot(ctx context.Context) error {
	a.flowMu.Lock()
	if a.flow != nil {
		a.flow.cancel()
		a.flow = nil
	}
	a.flowMu.Unlock()
	if err := user.Logout(); err != nil {
		return err
	}
	patch, err := json.Marshal(map[string]any{
		"userbot": map[string]any{"enable": false},
	})
	if err != nil {
		return fmt.Errorf("encode userbot logout settings: %w", err)
	}
	if _, err := a.svc.UpdateGroup(ctx, settings.GroupTelegram, patch); err != nil {
		return fmt.Errorf("failed to persist logout: %w", err)
	}
	return nil
}

func (a *App) RestartProcess() {
	a.RequestRestart()
}

func (a *App) getFlow(flowID string) (*loginFlow, error) {
	a.flowMu.Lock()
	defer a.flowMu.Unlock()
	if a.flow == nil || a.flow.id != flowID {
		return nil, errors.New("unknown login flow")
	}
	return a.flow, nil
}
