package api

import (
	"encoding/json"
	"net/http"
)

func (s *server) getSettings(w http.ResponseWriter, r *http.Request) {
	values, err := s.deps.Settings.PublicSettings()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, values)
}

func (s *server) updateSettings(w http.ResponseWriter, r *http.Request) {
	group := r.PathValue("group")
	var patch json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&patch); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	restart, err := s.deps.Settings.UpdateGroup(r.Context(), group, patch)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "settings_update_failed", err.Error())
		return
	}
	warnings := make([]string, 0)
	if group == "telegram" {
		if err := s.deps.Telegram.ApplyTelegram(r.Context()); err != nil {
			warnings = append(warnings, err.Error())
		}
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"restart_required": restart,
		"warnings":         warnings,
	})
}
