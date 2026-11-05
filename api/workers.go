package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/krau/SaveAny-Bot/client/worker"
	"github.com/krau/SaveAny-Bot/database"
)

const maskedWorkerToken = "__MASKED__"

type BotWorkerInput struct {
	Name    string `json:"name"`
	Token   string `json:"token"`
	Enabled bool   `json:"enabled"`
}

type BotWorkerView struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	SessionPath string `json:"session_path,omitempty"`
	TokenSet    bool   `json:"token_set"`
	State       string `json:"state"`
	Username    string `json:"username,omitempty"`
	Error       string `json:"error,omitempty"`
}

func botWorkerView(record *database.BotWorker) BotWorkerView {
	status := worker.StatusFor(record.ID)
	return BotWorkerView{
		ID:          record.ID,
		Name:        record.Name,
		Enabled:     record.Enabled,
		SessionPath: record.SessionPath,
		TokenSet:    record.Token != "",
		State:       status.State,
		Username:    status.Username,
		Error:       status.Error,
	}
}

func (s *server) listBotWorkers(w http.ResponseWriter, r *http.Request) {
	records, err := database.ListBotWorkers(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	out := make([]BotWorkerView, 0, len(records))
	for i := range records {
		out = append(out, botWorkerView(&records[i]))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"workers": out})
}

func (s *server) createBotWorker(w http.ResponseWriter, r *http.Request) {
	var in BotWorkerInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	name := strings.TrimSpace(in.Name)
	token := strings.TrimSpace(in.Token)
	if name == "" || len([]rune(name)) > 64 {
		WriteFieldErrors(w, http.StatusBadRequest, "invalid helper bot name", map[string]string{"name": "required, max 64 characters"})
		return
	}
	if token == "" {
		WriteFieldErrors(w, http.StatusBadRequest, "token is required", map[string]string{"token": "required"})
		return
	}
	sessionPath := worker.NewSessionPath(s.deps.DataDir)
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	record := &database.BotWorker{
		Name:        name,
		Token:       token,
		SessionPath: sessionPath,
		Enabled:     in.Enabled,
	}
	if err := database.CreateBotWorker(r.Context(), record); err != nil {
		WriteError(w, http.StatusConflict, "worker_create_failed", err.Error())
		return
	}
	if record.Enabled {
		if _, err := worker.StartWorker(s.deps.Ctx, record.ID); err != nil {
			WriteError(w, http.StatusInternalServerError, "worker_start_failed", err.Error())
			return
		}
	}
	WriteJSON(w, http.StatusCreated, botWorkerView(record))
}

func (s *server) updateBotWorker(w http.ResponseWriter, r *http.Request) {
	id, ok := botWorkerID(w, r)
	if !ok {
		return
	}
	record, err := database.GetBotWorker(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "worker_not_found", "helper bot not found")
		return
	}
	var in BotWorkerInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len([]rune(name)) > 64 {
		WriteFieldErrors(w, http.StatusBadRequest, "invalid helper bot name", map[string]string{"name": "required, max 64 characters"})
		return
	}
	fields := map[string]any{"name": name, "enabled": in.Enabled}
	if token := strings.TrimSpace(in.Token); token != "" && token != maskedWorkerToken && token != record.Token {
		worker.StopWorker(id)
		if err := worker.RemoveSession(record.SessionPath); err != nil {
			WriteError(w, http.StatusInternalServerError, "worker_session_cleanup_failed", err.Error())
			return
		}
		sessionPath := worker.NewSessionPath(s.deps.DataDir)
		if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		fields["token"] = token
		fields["session_path"] = sessionPath
	}
	if err := database.UpdateBotWorker(r.Context(), id, fields); err != nil {
		WriteError(w, http.StatusConflict, "worker_update_failed", err.Error())
		return
	}
	record, err = database.GetBotWorker(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if record.Enabled {
		if _, err := worker.StartWorker(s.deps.Ctx, id); err != nil {
			WriteError(w, http.StatusInternalServerError, "worker_start_failed", err.Error())
			return
		}
	} else {
		worker.StopWorker(id)
	}
	WriteJSON(w, http.StatusOK, botWorkerView(record))
}

func (s *server) deleteBotWorker(w http.ResponseWriter, r *http.Request) {
	id, ok := botWorkerID(w, r)
	if !ok {
		return
	}
	record, err := database.GetBotWorker(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "worker_not_found", "helper bot not found")
		return
	}
	worker.StopWorker(id)
	if err := database.DeleteBotWorker(r.Context(), id); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if err := worker.RemoveSession(record.SessionPath); err != nil {
		WriteError(w, http.StatusInternalServerError, "worker_session_cleanup_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) startBotWorker(w http.ResponseWriter, r *http.Request) {
	id, ok := botWorkerID(w, r)
	if !ok {
		return
	}
	record, err := database.GetBotWorker(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "worker_not_found", "helper bot not found")
		return
	}
	if !record.Enabled {
		WriteError(w, http.StatusBadRequest, "worker_disabled", "helper bot is disabled")
		return
	}
	status, err := worker.StartWorker(s.deps.Ctx, id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "worker_start_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, status)
}

func (s *server) stopBotWorker(w http.ResponseWriter, r *http.Request) {
	id, ok := botWorkerID(w, r)
	if !ok {
		return
	}
	if _, err := database.GetBotWorker(r.Context(), id); err != nil {
		WriteError(w, http.StatusNotFound, "worker_not_found", "helper bot not found")
		return
	}
	worker.StopWorker(id)
	WriteJSON(w, http.StatusOK, worker.StatusFor(id))
}

func (s *server) testBotWorker(w http.ResponseWriter, r *http.Request) {
	id, ok := botWorkerID(w, r)
	if !ok {
		return
	}
	if _, err := database.GetBotWorker(r.Context(), id); err != nil {
		WriteError(w, http.StatusNotFound, "worker_not_found", "helper bot not found")
		return
	}
	status := worker.StatusFor(id)
	if status.State != worker.StateRunning {
		WriteJSON(w, http.StatusOK, map[string]any{"ok": false, "error": status.Error, "state": status.State})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "username": status.Username})
}

func botWorkerID(w http.ResponseWriter, r *http.Request) (uint, bool) {
	id64, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid helper bot id")
		return 0, false
	}
	return uint(id64), true
}
