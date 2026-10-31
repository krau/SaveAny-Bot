package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
)

func (s *server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := database.GetAllUsers(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	out := make([]UserResponse, 0, len(users))
	for i := range users {
		out = append(out, userResponse(&users[i]))
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
	if body.ChatID == 0 {
		WriteFieldErrors(w, http.StatusBadRequest, "chat_id is required", map[string]string{"chat_id": "required"})
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
	if fields := applyUserPatch(candidate, body.UserPatch); len(fields) > 0 {
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
	if fields := applyUserPatch(user, body.UserPatch); len(fields) > 0 {
		WriteFieldErrors(w, http.StatusBadRequest, "invalid user fields", fields)
		return
	}
	if err := database.UpdateUser(r.Context(), user); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	_ = s.deps.Settings.ReloadConfig(r.Context())
	WriteJSON(w, http.StatusCreated, userResponse(user))
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
	if fields := applyUserPatch(user, body); len(fields) > 0 {
		WriteFieldErrors(w, http.StatusBadRequest, "invalid user fields", fields)
		return
	}
	if err := database.UpdateUser(r.Context(), user); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	_ = s.deps.Settings.ReloadConfig(r.Context())
	WriteJSON(w, http.StatusOK, userResponse(user))
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
	_ = s.deps.Settings.ReloadConfig(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func userResponse(user *database.User) UserResponse {
	names := make([]string, 0)
	if user.StoragesJSON != "" {
		_ = json.Unmarshal([]byte(user.StoragesJSON), &names)
	}
	return UserResponse{
		ID:               user.ID,
		ChatID:           user.ChatID,
		Enabled:          user.Enabled,
		Silent:           user.Silent,
		DefaultStorage:   user.DefaultStorage,
		DefaultDir:       user.DefaultDir,
		ApplyRule:        user.ApplyRule,
		FilenameStrategy: user.FilenameStrategy,
		FilenameTemplate: user.FilenameTemplate,
		ConflictStrategy: user.ConflictStrategy,
		WatchNotify:      user.WatchNotify,
		StorageFilter:    UserStorageFilter{Storages: names, Blacklist: user.Blacklist},
	}
}

func applyUserPatch(user *database.User, patch UserPatch) map[string]string {
	fields := make(map[string]string)
	if patch.Enabled != nil {
		user.Enabled = *patch.Enabled
	}
	if patch.Silent != nil {
		user.Silent = *patch.Silent
	}
	if patch.DefaultStorage != nil {
		if *patch.DefaultStorage != "" && config.C().GetStorageByName(*patch.DefaultStorage) == nil {
			fields["default_storage"] = "unknown storage"
		} else {
			user.DefaultStorage = *patch.DefaultStorage
		}
	}
	if patch.DefaultDir != nil {
		user.DefaultDir = *patch.DefaultDir
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
		for _, name := range patch.StorageFilter.Storages {
			if config.C().GetStorageByName(name) == nil {
				fields["storage_filter.storages"] = "unknown storage: " + name
				break
			}
		}
		raw, err := json.Marshal(patch.StorageFilter.Storages)
		if err != nil {
			fields["storage_filter.storages"] = err.Error()
		} else {
			user.StoragesJSON = string(raw)
			user.Blacklist = patch.StorageFilter.Blacklist
		}
	}
	return fields
}
