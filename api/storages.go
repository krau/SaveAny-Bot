package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/settings"
	"github.com/krau/SaveAny-Bot/storage"
)

var storageSecretKeys = map[string]struct{}{
	"password": {}, "token": {}, "secret": {}, "access_key_id": {}, "secret_access_key": {},
}

func maskStorageConfig(config map[string]any) map[string]any {
	out := make(map[string]any, len(config))
	for key, value := range config {
		if _, secret := storageSecretKeys[key]; secret {
			if s, ok := value.(string); ok && s != "" {
				out[key] = "__MASKED__"
				continue
			}
		}
		out[key] = value
	}
	return out
}

func storageResponse(record *database.StorageRecord, initErr string) StorageRecord {
	var config map[string]any
	if record.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(record.ConfigJSON), &config); err != nil {
			log.Warnf("Invalid storage config for %s: %v", record.Name, err)
		}
	}
	if config == nil {
		config = make(map[string]any)
	}
	return StorageRecord{
		ID:     record.ID,
		Name:   record.Name,
		Type:   record.Type,
		Enable: record.Enable,
		Config: maskStorageConfig(config),
		Error:  initErr,
	}
}

func (s *server) listStorages(w http.ResponseWriter, r *http.Request) {
	records, err := database.ListStorageRecords(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	initErrors := storage.LastErrors()
	out := make([]StorageRecord, 0, len(records))
	for i := range records {
		out = append(out, storageResponse(&records[i], initErrors[records[i].Name]))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"storages": out})
}

func (s *server) createStorage(w http.ResponseWriter, r *http.Request) {
	var in StorageInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	record, err := s.deps.Settings.CreateStorage(r.Context(), settings.StorageInput{
		Name: in.Name, Type: in.Type, Enable: in.Enable, Config: in.Config,
	})
	if err != nil {
		if errors.Is(err, settings.ErrStorageExists) {
			WriteError(w, http.StatusConflict, "storage_exists", err.Error())
			return
		}
		WriteError(w, http.StatusBadRequest, "storage_create_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, storageResponse(record, ""))
}

func (s *server) updateStorage(w http.ResponseWriter, r *http.Request) {
	id64, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid storage id")
		return
	}
	var in StorageInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	record, err := s.deps.Settings.UpdateStorage(r.Context(), uint(id64), settings.StorageInput{
		Name: in.Name, Type: in.Type, Enable: in.Enable, Config: in.Config,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "storage_update_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, storageResponse(record, ""))
}

func (s *server) deleteStorage(w http.ResponseWriter, r *http.Request) {
	id64, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid storage id")
		return
	}
	if err := s.deps.Settings.DeleteStorage(r.Context(), uint(id64)); err != nil {
		if errors.Is(err, settings.ErrStorageInUse) {
			WriteError(w, http.StatusConflict, "storage_in_use", err.Error())
			return
		}
		WriteError(w, http.StatusBadRequest, "storage_delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) testStorage(w http.ResponseWriter, r *http.Request) {
	id64, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid storage id")
		return
	}
	if err := s.deps.Settings.TestStorage(r.Context(), uint(id64)); err != nil {
		WriteJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}
