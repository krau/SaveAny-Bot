package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/storagefilter"
)

func (s *server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := database.GetAllUsers(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	names, err := storageNamesByID(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	out := make([]UserResponse, 0, len(users))
	for i := range users {
		out = append(out, userResponse(&users[i], names))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (s *server) createUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChatID int64 `json:"chat_id"`
		UserPatch
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if body.ChatID <= 0 {
		WriteFieldErrors(w, http.StatusBadRequest, "chat_id must be a positive Telegram user ID", map[string]string{"chat_id": "invalid"})
		return
	}
	if _, err := database.GetUserByChatID(r.Context(), body.ChatID); err == nil {
		WriteError(w, http.StatusConflict, "user_exists", "user already exists")
		return
	} else if !database.IsNotFound(err) {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	candidate := &database.User{ChatID: body.ChatID, Enabled: true}
	if fields := applyUserPatch(r.Context(), candidate, body.UserPatch); len(fields) > 0 {
		WriteFieldErrors(w, http.StatusBadRequest, "invalid user fields", fields)
		return
	}
	if err := database.CreateUser(r.Context(), body.ChatID); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	user, err := database.GetUserByChatID(r.Context(), body.ChatID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if fields := applyUserPatch(r.Context(), user, body.UserPatch); len(fields) > 0 {
		WriteFieldErrors(w, http.StatusBadRequest, "invalid user fields", fields)
		return
	}
	if err := database.UpdateUser(r.Context(), user); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if err := s.deps.Settings.ReloadConfig(r.Context()); err != nil {
		log.FromContext(r.Context()).Warnf("Failed to reload settings after user creation: %v", err)
	}
	names, err := storageNamesByID(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, userResponse(user, names))
}

func (s *server) updateUser(w http.ResponseWriter, r *http.Request) {
	id64, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return
	}
	user, err := database.GetUserByID(r.Context(), uint(id64))
	if err != nil {
		WriteError(w, http.StatusNotFound, "user_not_found", "user not found")
		return
	}
	var body UserPatch
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if fields := applyUserPatch(r.Context(), user, body); len(fields) > 0 {
		WriteFieldErrors(w, http.StatusBadRequest, "invalid user fields", fields)
		return
	}
	if err := database.UpdateUser(r.Context(), user); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if err := s.deps.Settings.ReloadConfig(r.Context()); err != nil {
		log.FromContext(r.Context()).Warnf("Failed to reload settings after user update: %v", err)
	}
	names, err := storageNamesByID(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, userResponse(user, names))
}

func (s *server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id64, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return
	}
	user, err := database.GetUserByID(r.Context(), uint(id64))
	if err != nil {
		WriteError(w, http.StatusNotFound, "user_not_found", "user not found")
		return
	}
	if err := database.DeleteUser(r.Context(), user); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if err := s.deps.Settings.ReloadConfig(r.Context()); err != nil {
		log.FromContext(r.Context()).Warnf("Failed to reload settings after user deletion: %v", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// storageNamesByID resolves names for both enabled and disabled storage
// records so references remain readable after a storage is disabled.
func storageNamesByID(ctx context.Context) (map[uint]string, error) {
	records, err := database.ListStorageRecords(ctx)
	if err != nil {
		return nil, err
	}
	names := make(map[uint]string, len(records))
	for _, record := range records {
		names[record.ID] = record.Name
	}
	return names, nil
}

func userResponse(user *database.User, namesByID map[uint]string) UserResponse {
	ids := make([]uint, 0)
	if user.StorageFilterIDsJSON != "" {
		if err := json.Unmarshal([]byte(user.StorageFilterIDsJSON), &ids); err != nil {
			log.Warnf("Invalid storage filter for user %d: %v", user.ChatID, err)
		}
	}
	if ids == nil {
		ids = make([]uint, 0)
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, namesByID[id])
	}
	return UserResponse{
		ID:                 user.ID,
		ChatID:             user.ChatID,
		Enabled:            user.Enabled,
		Silent:             user.Silent,
		DefaultStorageID:   user.DefaultStorageID,
		DefaultStorageName: namesByID[user.DefaultStorageID],
		DefaultDirID:       user.DefaultDirID,
		ApplyRule:          user.ApplyRule,
		FilenameStrategy:   user.FilenameStrategy,
		FilenameTemplate:   user.FilenameTemplate,
		ConflictStrategy:   user.ConflictStrategy,
		WatchNotify:        user.WatchNotify,
		StorageFilter:      UserStorageFilter{StorageIDs: ids, StorageNames: names, Mode: user.StorageFilterMode.Normalize()},
	}
}

func applyUserPatch(ctx context.Context, user *database.User, patch UserPatch) map[string]string {
	fields := make(map[string]string)
	if patch.Enabled != nil {
		user.Enabled = *patch.Enabled
	}
	if patch.Silent != nil {
		user.Silent = *patch.Silent
	}
	if patch.DefaultStorageID != nil {
		if *patch.DefaultStorageID != 0 {
			if _, err := database.GetStorageRecord(ctx, *patch.DefaultStorageID); err != nil {
				fields["default_storage_id"] = "unknown storage"
			}
		}
		if _, exists := fields["default_storage_id"]; !exists {
			user.DefaultStorageID = *patch.DefaultStorageID
		}
	}
	if patch.DefaultDirID != nil {
		user.DefaultDirID = *patch.DefaultDirID
	}
	if patch.ApplyRule != nil {
		user.ApplyRule = *patch.ApplyRule
	}
	if patch.FilenameStrategy != nil {
		user.FilenameStrategy = *patch.FilenameStrategy
	}
	if patch.FilenameTemplate != nil {
		user.FilenameTemplate = *patch.FilenameTemplate
	}
	if patch.ConflictStrategy != nil {
		user.ConflictStrategy = *patch.ConflictStrategy
	}
	if patch.WatchNotify != nil {
		user.WatchNotify = *patch.WatchNotify
	}
	if patch.StorageFilter != nil {
		for _, id := range patch.StorageFilter.StorageIDs {
			if _, err := database.GetStorageRecord(ctx, id); err != nil {
				fields["storage_filter.storage_ids"] = "unknown storage"
				break
			}
		}
		mode := patch.StorageFilter.Mode
		if mode == "" {
			mode = storagefilter.Blacklist
		} else if !mode.Valid() {
			fields["storage_filter.mode"] = "must be blacklist or whitelist"
		}
		storageIDs := patch.StorageFilter.StorageIDs
		if storageIDs == nil {
			storageIDs = make([]uint, 0)
		}
		raw, err := json.Marshal(storageIDs)
		if err != nil {
			fields["storage_filter.storage_ids"] = err.Error()
		} else {
			user.StorageFilterIDsJSON = string(raw)
			user.StorageFilterMode = mode
		}
	}
	return fields
}
