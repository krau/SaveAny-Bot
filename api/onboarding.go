package api

import (
	"net/http"
	"strings"

	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/settings"
	"github.com/krau/SaveAny-Bot/storage"
)

type OnboardingSteps struct {
	BotConfigured bool `json:"bot_configured"`
	StorageReady  bool `json:"storage_ready"`
	UserReady     bool `json:"user_ready"`
	BotRunning    bool `json:"bot_running"`
	FirstTask     bool `json:"first_task"`
}

type OnboardingLegacy struct {
	Imported   bool   `json:"imported"`
	ImportedAt string `json:"imported_at,omitempty"`
}

type OnboardingStatus struct {
	Complete bool             `json:"complete"`
	Steps    OnboardingSteps  `json:"steps"`
	BotState string           `json:"bot_state"`
	Legacy   OnboardingLegacy `json:"legacy"`
}

// onboarding reports the minimum configuration needed before the bot can
// accept a task. It intentionally exposes no credentials.
func (s *server) onboarding(w http.ResponseWriter, r *http.Request) {
	cfg := s.deps.Settings.Config()
	records, err := database.ListStorageRecords(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	users, err := database.GetAllUsers(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	_, taskTotal, err := database.ListTaskRecords(r.Context(), database.TaskFilter{Limit: 1})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	storageReady := false
	initErrors := storage.LastErrors()
	for _, record := range records {
		if !record.Enable {
			continue
		}
		if message, failed := initErrors[record.Name]; failed && message != "" {
			continue
		}
		storageReady = true
		break
	}
	userReady := false
	for _, user := range users {
		if user.Enabled {
			userReady = true
			break
		}
	}

	botToken := strings.TrimSpace(cfg.Telegram.Token) != ""
	botStatus := s.deps.Telegram.Status(r.Context())
	importedAt, imported := settings.LegacyImportTime(r.Context())
	legacy := OnboardingLegacy{Imported: imported}
	if !importedAt.IsZero() {
		legacy.ImportedAt = importedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}

	status := OnboardingStatus{
		Complete: botToken && storageReady && userReady,
		Steps: OnboardingSteps{
			BotConfigured: botToken,
			StorageReady:  storageReady,
			UserReady:     userReady,
			BotRunning:    botStatus.Bot.State == "running",
			FirstTask:     taskTotal > 0,
		},
		BotState: botStatus.Bot.State,
		Legacy:   legacy,
	}
	WriteJSON(w, http.StatusOK, status)
}
