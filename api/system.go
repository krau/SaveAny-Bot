package api

import (
	"net/http"
	"time"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
)

func (s *server) systemInfo(w http.ResponseWriter, r *http.Request) {
	cfg := s.deps.Settings.Config()
	WriteJSON(w, http.StatusOK, map[string]any{
		"version":        config.Version,
		"commit":         config.GitCommit,
		"build_time":     config.BuildTime,
		"docker":         config.Docker == "true",
		"uptime_seconds": int64(time.Since(s.startedAt).Seconds()),
		"data_dir":       s.deps.DataDir,
		"db_path":        s.deps.DBPath,
		"workers":        cfg.Workers,
		"queued_tasks":   len(core.GetQueuedTasks(r.Context())),
		"running_tasks":  len(core.GetRunningTasks(r.Context())),
	})
}

func (s *server) restartSystem(w http.ResponseWriter, r *http.Request) {
	force, err := decodeForce(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if !force && core.ActiveTaskCount() > 0 {
		WriteError(w, http.StatusConflict, "tasks_running", "tasks are running; use {\"force\":true} to restart anyway")
		return
	}
	WriteJSON(w, http.StatusAccepted, map[string]any{"restarting": true})
	go func() {
		time.Sleep(200 * time.Millisecond)
		s.deps.Telegram.RestartProcess()
	}()
}
