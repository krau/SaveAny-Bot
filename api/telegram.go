package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func decodeForce(r *http.Request) bool {
	var body struct {
		Force bool `json:"force"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)
	return body.Force
}

func (s *server) telegramStatus(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, s.deps.Telegram.Status(r.Context()))
}

func (s *server) startBot(w http.ResponseWriter, r *http.Request) {
	cfg := s.deps.Settings.Config()
	if strings.TrimSpace(cfg.Telegram.Token) == "" {
		WriteError(w, http.StatusBadRequest, "missing_bot_token", "bot token is not configured")
		return
	}
	if err := s.deps.Telegram.StartBot(r.Context()); err != nil {
		WriteError(w, http.StatusBadRequest, "bot_start_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusAccepted, map[string]any{"state": "starting"})
}

func (s *server) stopBot(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.Telegram.StopBot(r.Context(), decodeForce(r)); err != nil {
		WriteError(w, http.StatusConflict, "bot_stop_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) restartBot(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.Telegram.RestartBot(r.Context(), decodeForce(r)); err != nil {
		WriteError(w, http.StatusConflict, "bot_restart_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusAccepted, map[string]any{"state": "starting"})
}

func (s *server) startUserbotLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	flowID, err := s.deps.Telegram.StartUserbotLogin(r.Context(), strings.TrimSpace(body.Phone))
	if err != nil {
		WriteError(w, http.StatusConflict, "userbot_login_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusAccepted, map[string]any{"flow_id": flowID})
}

func (s *server) submitUserbotCode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if err := s.deps.Telegram.SubmitUserbotCode(r.Context(), r.PathValue("flow"), strings.TrimSpace(body.Code)); err != nil {
		WriteError(w, http.StatusConflict, "userbot_login_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

func (s *server) submitUserbotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if err := s.deps.Telegram.SubmitUserbotPassword(r.Context(), r.PathValue("flow"), body.Password); err != nil {
		WriteError(w, http.StatusConflict, "userbot_login_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

func (s *server) cancelUserbotLogin(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.Telegram.CancelUserbotLogin(r.Context(), r.PathValue("flow")); err != nil {
		WriteError(w, http.StatusConflict, "userbot_login_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) logoutUserbot(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.Telegram.LogoutUserbot(r.Context()); err != nil {
		WriteError(w, http.StatusInternalServerError, "userbot_logout_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
