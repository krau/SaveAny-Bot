package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/rule"
)

type userDirResponse struct {
	ID          uint   `json:"id"`
	StorageID   uint   `json:"storage_id"`
	StorageName string `json:"storage_name,omitempty"`
	Path        string `json:"path"`
}

type userRuleResponse struct {
	ID          uint   `json:"id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	StorageID   uint   `json:"storage_id"`
	StorageName string `json:"storage_name,omitempty"`
	DirPath     string `json:"dir_path"`
}

type userWatchResponse struct {
	ID            uint   `json:"id"`
	ChatID        int64  `json:"chat_id"`
	MessageFilter string `json:"message_filter,omitempty"`
}

func (s *server) requireUser(w http.ResponseWriter, r *http.Request) (*database.User, bool) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return nil, false
	}
	user, err := database.GetUserByID(r.Context(), uint(id))
	if err != nil {
		WriteError(w, http.StatusNotFound, "user_not_found", "user not found")
		return nil, false
	}
	return user, true
}

func userDirsResponse(dirs []database.Dir, namesByID map[uint]string) []userDirResponse {
	out := make([]userDirResponse, 0, len(dirs))
	for _, dir := range dirs {
		out = append(out, userDirResponse{
			ID:          dir.ID,
			StorageID:   dir.StorageID,
			StorageName: namesByID[dir.StorageID],
			Path:        dir.Path,
		})
	}
	return out
}

func (s *server) listUserDirs(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	dirs, err := database.GetUserDirs(r.Context(), user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	names, err := storageNamesByID(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"dirs": userDirsResponse(dirs, names)})
}

func (s *server) createUserDir(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		StorageID uint   `json:"storage_id"`
		Path      string `json:"path"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	body.Path = strings.TrimSpace(body.Path)
	if body.StorageID == 0 || body.Path == "" {
		WriteFieldErrors(w, http.StatusBadRequest, "storage_id and path are required", map[string]string{
			"storage_id": "required",
			"path":       "required",
		})
		return
	}
	if _, err := database.GetStorageRecord(r.Context(), body.StorageID); err != nil {
		WriteFieldErrors(w, http.StatusBadRequest, "unknown storage", map[string]string{"storage_id": "unknown storage"})
		return
	}
	existing, err := database.GetDirsByUserIDAndStorageID(r.Context(), user.ID, body.StorageID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	for _, dir := range existing {
		if dir.Path == body.Path {
			WriteError(w, http.StatusConflict, "dir_exists", "directory already exists")
			return
		}
	}
	if err := database.CreateDirForUser(r.Context(), user.ID, body.StorageID, body.Path); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	dirs, err := database.GetUserDirs(r.Context(), user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	names, err := storageNamesByID(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	for _, dir := range dirs {
		if dir.StorageID == body.StorageID && dir.Path == body.Path {
			WriteJSON(w, http.StatusCreated, userDirResponse{
				ID:          dir.ID,
				StorageID:   dir.StorageID,
				StorageName: names[dir.StorageID],
				Path:        dir.Path,
			})
			return
		}
	}
	WriteError(w, http.StatusInternalServerError, "internal_error", "directory was not persisted")
}

func (s *server) deleteUserDir(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	dirID, err := strconv.ParseUint(r.PathValue("dirID"), 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid directory id")
		return
	}
	if err := database.DeleteDirByID(r.Context(), uint(dirID), user.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if user.DefaultDirID == uint(dirID) {
		user.DefaultDirID = 0
		if err := database.UpdateUser(r.Context(), user); err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func userRulesResponse(rules []database.Rule, namesByID map[uint]string) []userRuleResponse {
	out := make([]userRuleResponse, 0, len(rules))
	for _, rule := range rules {
		out = append(out, userRuleResponse{
			ID:          rule.ID,
			Type:        rule.Type,
			Content:     rule.Content,
			StorageID:   rule.StorageID,
			StorageName: namesByID[rule.StorageID],
			DirPath:     rule.DirPath,
		})
	}
	return out
}

func validateRuleContent(ruleType, content string) error {
	switch ruleType {
	case rule.FileNameRegex.String(), rule.MessageRegex.String():
		if _, err := regexp.Compile(content); err != nil {
			return fmt.Errorf("invalid regular expression: %w", err)
		}
	case rule.IsAlbum.String():
		if _, err := strconv.ParseBool(content); err != nil {
			return errors.New("IS-ALBUM content must be true or false")
		}
	default:
		return fmt.Errorf("unsupported rule type: %s", ruleType)
	}
	return nil
}

func normalizeRuleType(raw string) (string, error) {
	for _, typ := range rule.Values() {
		if strings.EqualFold(strings.TrimSpace(raw), typ.String()) {
			return typ.String(), nil
		}
	}
	return "", fmt.Errorf("unsupported rule type: %s", raw)
}

func (s *server) listUserRules(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	rules, err := database.GetRulesByUserID(r.Context(), user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	names, err := storageNamesByID(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"rules": userRulesResponse(rules, names)})
}

func (s *server) createUserRule(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Type      string `json:"type"`
		Content   string `json:"content"`
		StorageID uint   `json:"storage_id"`
		DirPath   string `json:"dir_path"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	ruleType, err := normalizeRuleType(body.Type)
	if err != nil {
		WriteFieldErrors(w, http.StatusBadRequest, err.Error(), map[string]string{"type": err.Error()})
		return
	}
	body.Content = strings.TrimSpace(body.Content)
	if err := validateRuleContent(ruleType, body.Content); err != nil {
		WriteFieldErrors(w, http.StatusBadRequest, err.Error(), map[string]string{"content": err.Error()})
		return
	}
	if body.StorageID != 0 {
		if _, err := database.GetStorageRecord(r.Context(), body.StorageID); err != nil {
			WriteFieldErrors(w, http.StatusBadRequest, "unknown storage", map[string]string{"storage_id": "unknown storage"})
			return
		}
	}
	record := &database.Rule{
		UserID:    user.ID,
		Type:      ruleType,
		Content:   body.Content,
		StorageID: body.StorageID,
		DirPath:   strings.TrimSpace(body.DirPath),
	}
	if err := database.CreateRule(r.Context(), record); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	names, err := storageNamesByID(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, userRuleResponse{
		ID: record.ID, Type: record.Type, Content: record.Content,
		StorageID: record.StorageID, StorageName: names[record.StorageID],
		DirPath: record.DirPath,
	})
}

func (s *server) deleteUserRule(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ruleID, err := strconv.ParseUint(r.PathValue("ruleID"), 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid rule id")
		return
	}
	if err := database.DeleteRule(r.Context(), uint(ruleID), user.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) importUserRulePresets(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		StorageID uint   `json:"storage_id"`
		BasePath  string `json:"base_path"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if body.StorageID != 0 {
		if _, err := database.GetStorageRecord(r.Context(), body.StorageID); err != nil {
			WriteFieldErrors(w, http.StatusBadRequest, "unknown storage", map[string]string{"storage_id": "unknown storage"})
			return
		}
	}
	presets := rule.PresetCategories(strings.TrimSpace(body.BasePath))
	created := 0
	for _, preset := range presets {
		record := &database.Rule{
			UserID:    user.ID,
			Type:      rule.FileNameRegex.String(),
			Content:   preset.Regex,
			StorageID: body.StorageID,
			DirPath:   preset.Dir,
		}
		if err := database.CreateRule(r.Context(), record); err != nil {
			continue
		}
		created++
	}
	if created == 0 {
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create preset rules")
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"created": created})
}

func userWatchesResponse(chats []database.WatchChat) []userWatchResponse {
	out := make([]userWatchResponse, 0, len(chats))
	for _, chat := range chats {
		out = append(out, userWatchResponse{ID: chat.ID, ChatID: chat.ChatID, MessageFilter: chat.MessageFilter})
	}
	return out
}

func normalizeWatchFilter(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	filterType, filterData, ok := strings.Cut(raw, ":")
	if !ok || !strings.EqualFold(filterType, "msgre") || filterData == "" {
		return "", errors.New("filter must be msgre:<regular expression>")
	}
	if _, err := regexp.Compile(filterData); err != nil {
		return "", fmt.Errorf("invalid filter expression: %w", err)
	}
	return "msgre:" + filterData, nil
}

func (s *server) listUserWatches(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"watches": userWatchesResponse(user.WatchChats)})
}

func (s *server) createUserWatch(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		ChatID int64  `json:"chat_id"`
		Chat   string `json:"chat"`
		Filter string `json:"filter"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if body.ChatID == 0 && strings.TrimSpace(body.Chat) != "" {
		chatID, err := resolveChatID(r.Context(), strings.TrimSpace(body.Chat))
		if err != nil {
			WriteFieldErrors(w, http.StatusBadRequest, "invalid chat", map[string]string{"chat": err.Error()})
			return
		}
		body.ChatID = chatID
	}
	if body.ChatID == 0 {
		WriteFieldErrors(w, http.StatusBadRequest, "chat is required", map[string]string{"chat": "required"})
		return
	}
	filter, err := normalizeWatchFilter(body.Filter)
	if err != nil {
		WriteFieldErrors(w, http.StatusBadRequest, err.Error(), map[string]string{"filter": err.Error()})
		return
	}
	watching, err := user.WatchingChat(r.Context(), body.ChatID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if watching {
		WriteError(w, http.StatusConflict, "watch_exists", "chat is already watched")
		return
	}
	if err := user.WatchChat(r.Context(), database.WatchChat{
		UserID:        user.ID,
		ChatID:        body.ChatID,
		MessageFilter: filter,
	}); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	refreshed, err := database.GetUserByID(r.Context(), user.ID)
	if err == nil {
		user = refreshed
	}
	WriteJSON(w, http.StatusCreated, userWatchesResponse(user.WatchChats))
}

func (s *server) deleteUserWatch(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	chatID, err := strconv.ParseInt(r.PathValue("chatID"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid chat id")
		return
	}
	if err := user.UnwatchChat(r.Context(), chatID); err != nil {
		WriteError(w, http.StatusNotFound, "watch_not_found", "watched chat not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
