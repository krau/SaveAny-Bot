package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/ruleutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	tftask "github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/queue"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

// Request/Response types
type CreateTaskRequest struct {
	TelegramURL string `json:"telegram_url"`
	StorageName string `json:"storage_name,omitempty"`
	DirPath     string `json:"dir_path,omitempty"`
	UserID      int64  `json:"user_id"`
}

type CreateTaskResponse struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

type TaskStatusResponse struct {
	TaskID      string    `json:"task_id"`
	Status      string    `json:"status"` // queued, running, completed, failed, canceled
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	Error       string    `json:"error,omitempty"`
	Downloaded  int64     `json:"downloaded,omitempty"`   // Bytes downloaded
	Total       int64     `json:"total,omitempty"`        // Total bytes
	ProgressPct float64   `json:"progress_pct,omitempty"` // Progress percentage (0-100)
}

type ListTasksResponse struct {
	Queued  []TaskInfo `json:"queued"`
	Running []TaskInfo `json:"running"`
}

type TaskInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Task tracking
var (
	taskStatuses   = make(map[string]*taskStatus)
	taskStatusesMu sync.RWMutex
)

type taskStatus struct {
	ID          string
	Status      string
	Title       string
	CreatedAt   time.Time
	Error       string
	Downloaded  atomic.Int64 // Use atomic for lock-free updates
	Total       atomic.Int64 // Use atomic for lock-free updates
	ProgressPct uint64       // Store as uint64 bits of float64 for atomic access
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.TelegramURL == "" {
		respondError(w, "telegram_url is required", http.StatusBadRequest)
		return
	}
	if req.UserID <= 0 {
		respondError(w, "user_id is required and must be positive", http.StatusBadRequest)
		return
	}

	logger := log.FromContext(r.Context()).WithPrefix("api")

	// Get user from database
	userDB, err := database.GetUserByChatID(r.Context(), req.UserID)
	if err != nil {
		logger.Errorf("Failed to get user: %v", err)
		respondError(w, "user not found", http.StatusBadRequest)
		return
	}

	// Get storage
	var stor storage.Storage
	if req.StorageName != "" {
		stor, err = storage.GetStorageByUserIDAndName(r.Context(), req.UserID, req.StorageName)
		if err != nil {
			logger.Errorf("Failed to get storage: %v", err)
			respondError(w, "storage not found", http.StatusBadRequest)
			return
		}
	} else {
		// Use first available storage for the user
		storages := storage.GetUserStorages(r.Context(), req.UserID)
		if len(storages) == 0 {
			respondError(w, "no storage available for user", http.StatusBadRequest)
			return
		}
		stor = storages[0]
	}

	// Parse Telegram URL
	botCtx := bot.ExtContext()
	if botCtx == nil {
		respondError(w, "bot not initialized", http.StatusInternalServerError)
		return
	}

	linkUrl, err := url.Parse(req.TelegramURL)
	if err != nil {
		logger.Errorf("Failed to parse URL: %v", err)
		respondError(w, "invalid telegram URL format", http.StatusBadRequest)
		return
	}

	chatID, msgID, err := tgutil.ParseMessageLink(botCtx, req.TelegramURL)
	if err != nil {
		logger.Errorf("Failed to parse Telegram URL: %v", err)
		respondError(w, "invalid telegram URL format", http.StatusBadRequest)
		return
	}

	// Get message from Telegram
	msg, err := tgutil.GetMessageByID(botCtx, chatID, msgID)
	if err != nil {
		logger.Errorf("Failed to get message: %v", err)
		respondError(w, "failed to retrieve message", http.StatusBadRequest)
		return
	}

	// Check if message has media
	media, ok := msg.GetMedia()
	if !ok {
		respondError(w, "message has no media", http.StatusBadRequest)
		return
	}

	// Collect files - handle both single and grouped messages
	files := make([]tfile.TGFileMessage, 0)

	// Check for grouped messages (media group)
	groupID, isGroup := msg.GetGroupedID()
	if isGroup && groupID != 0 && !linkUrl.Query().Has("single") {
		// Handle media group
		gmsgs, err := tgutil.GetGroupedMessages(botCtx, chatID, msg)
		if err != nil {
			logger.Errorf("Failed to get grouped messages: %v", err)
			// Fall back to single message
			file, err := createTGFileWithMedia(botCtx, msg, media, userDB)
			if err != nil {
				logger.Errorf("Failed to create TGFile: %v", err)
				respondError(w, "invalid message format", http.StatusBadRequest)
				return
			}
			files = append(files, file)
		} else {
			// Process all messages in the group
			for _, gmsg := range gmsgs {
				if gmsg.Media == nil {
					continue
				}
				gMedia, ok := gmsg.GetMedia()
				if !ok {
					continue
				}
				file, err := createTGFileWithMedia(botCtx, gmsg, gMedia, userDB)
				if err != nil {
					logger.Warnf("Failed to create TGFile for grouped message: %v", err)
					continue
				}
				files = append(files, file)
			}
		}
	} else {
		// Single message
		file, err := createTGFileWithMedia(botCtx, msg, media, userDB)
		if err != nil {
			logger.Errorf("Failed to create TGFile: %v", err)
			respondError(w, "invalid message format", http.StatusBadRequest)
			return
		}
		files = append(files, file)
	}

	if len(files) == 0 {
		respondError(w, "no savable files found", http.StatusBadRequest)
		return
	}

	// Create tasks for all files
	taskIDs := make([]string, 0, len(files))
	baseDirPath := req.DirPath
	if baseDirPath == "" {
		baseDirPath = "/"
	}

	// Create context with bot extension
	injectCtx := tgutil.ExtWithContext(r.Context(), botCtx)

	// Apply storage rules if enabled for the user
	useRule := userDB.ApplyRule && userDB.Rules != nil

	for _, tgFile := range files {
		// Determine storage and directory path for this specific file
		fileStor := stor
		dirPath := baseDirPath
		
		// Apply rules if enabled
		if useRule {
			matched, matchedStorName, matchedDirPath := ruleutil.ApplyRule(injectCtx, userDB.Rules, ruleutil.NewInput(tgFile))
			if matched {
				// Rule matched, apply overrides
				if matchedDirPath != "" && matchedDirPath != "{{album}}" {
					dirPath = matchedDirPath.String()
				}
				if matchedStorName.Usable() {
					var err error
					fileStor, err = storage.GetStorageByUserIDAndName(injectCtx, userDB.ChatID, matchedStorName.String())
					if err != nil {
						logger.Errorf("Failed to get storage from rule: %v", err)
						// Fall back to original storage
						fileStor = stor
					}
				}
			}
		}

		storagePath := fileStor.JoinStoragePath(path.Join(dirPath, tgFile.Name()))
		taskID := xid.New().String()

		task, err := tftask.NewTGFileTask(taskID, injectCtx, tgFile, fileStor, storagePath, &apiProgressTracker{
			taskID: taskID,
		})
		if err != nil {
			logger.Errorf("Failed to create task: %v", err)
			respondError(w, "failed to create task", http.StatusInternalServerError)
			return
		}

		// Track task status
		trackTask(taskID, task.Title(), "queued")

		// Add task to queue
		if err := core.AddTask(injectCtx, task); err != nil {
			logger.Errorf("Failed to add task: %v", err)
			updateTaskStatus(taskID, "failed", err.Error())
			respondError(w, "failed to add task to queue", http.StatusInternalServerError)
			return
		}

		taskIDs = append(taskIDs, taskID)
	}

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Return first task ID for single file, or all task IDs for media group
	if len(taskIDs) == 1 {
		json.NewEncoder(w).Encode(CreateTaskResponse{
			TaskID:  taskIDs[0],
			Message: "task created successfully",
		})
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"task_ids": taskIDs,
			"message":  fmt.Sprintf("%d tasks created successfully", len(taskIDs)),
		})
	}
}

// createTGFileWithMedia creates a TGFile with proper filename handling using user's strategy
func createTGFileWithMedia(botCtx *ext.Context, msg *tg.Message, media tg.MessageMediaClass, userDB *database.User) (tfile.TGFileMessage, error) {
	// Use the same filename generation logic as bot handlers
	opts := []tfile.TGFileOption{tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*msg))}
	return tfile.FromMediaMessage(media, botCtx.Raw, msg, opts...)
}

func handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		respondError(w, "task_id is required", http.StatusBadRequest)
		return
	}

	taskStatusesMu.RLock()
	status, exists := taskStatuses[taskID]
	taskStatusesMu.RUnlock()

	if !exists {
		respondError(w, "task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TaskStatusResponse{
		TaskID:      status.ID,
		Status:      status.Status,
		Title:       status.Title,
		CreatedAt:   status.CreatedAt,
		Error:       status.Error,
		Downloaded:  status.Downloaded.Load(),
		Total:       status.Total.Load(),
		ProgressPct: math.Float64frombits(atomic.LoadUint64((*uint64)(&status.ProgressPct))),
	})
}

func handleListTasks(w http.ResponseWriter, r *http.Request) {
	queued := core.GetQueuedTasks(r.Context())
	running := core.GetRunningTasks(r.Context())

	response := ListTasksResponse{
		Queued:  convertTaskInfos(queued),
		Running: convertTaskInfos(running),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleCancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		respondError(w, "task_id is required", http.StatusBadRequest)
		return
	}

	if err := core.CancelTask(r.Context(), taskID); err != nil {
		log.FromContext(r.Context()).Errorf("Failed to cancel task %s: %v", taskID, err)
		respondError(w, "failed to cancel task", http.StatusInternalServerError)
		return
	}

	updateTaskStatus(taskID, "canceled", "")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "task canceled"})
}

// Helper functions

func respondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func trackTask(taskID, title, status string) {
	taskStatusesMu.Lock()
	defer taskStatusesMu.Unlock()
	taskStatuses[taskID] = &taskStatus{
		ID:        taskID,
		Status:    status,
		Title:     title,
		CreatedAt: time.Now(),
	}
}

func updateTaskStatus(taskID, status, errorMsg string) {
	taskStatusesMu.Lock()
	defer taskStatusesMu.Unlock()
	if ts, exists := taskStatuses[taskID]; exists {
		ts.Status = status
		ts.Error = errorMsg
	}
}

func convertTaskInfos(tasks []queue.TaskInfo) []TaskInfo {
	result := make([]TaskInfo, len(tasks))
	for i, t := range tasks {
		result[i] = TaskInfo{
			ID:    t.ID,
			Title: t.Title,
		}
	}
	return result
}

// apiProgressTracker implements tftask.ProgressTracker for API tasks
type apiProgressTracker struct {
	taskID string
}

func (a *apiProgressTracker) OnStart(ctx context.Context, info tftask.TaskInfo) {
	updateTaskStatus(a.taskID, "running", "")
}

func (a *apiProgressTracker) OnProgress(ctx context.Context, info tftask.TaskInfo, downloaded int64, total int64) {
	// Use atomic operations to avoid mutex locks for better performance
	// OnProgress is called very frequently during downloads
	taskStatusesMu.RLock()
	ts, exists := taskStatuses[a.taskID]
	taskStatusesMu.RUnlock()
	
	if exists {
		ts.Downloaded.Store(downloaded)
		ts.Total.Store(total)
		if total > 0 {
			progressPct := float64(downloaded) / float64(total) * 100.0
			atomic.StoreUint64((*uint64)(&ts.ProgressPct), math.Float64bits(progressPct))
		}
	}
}

func (a *apiProgressTracker) OnDone(ctx context.Context, info tftask.TaskInfo, err error) {
	if err != nil {
		updateTaskStatus(a.taskID, "failed", err.Error())
		sendWebhook(a.taskID, "failed", err.Error())
	} else {
		updateTaskStatus(a.taskID, "completed", "")
		sendWebhook(a.taskID, "completed", "")
	}
}

// sendWebhook sends a callback to the configured webhook URL
func sendWebhook(taskID, status, errorMsg string) {
	cfg := config.C()
	if cfg.API.WebhookURL == "" {
		return
	}

	taskStatusesMu.RLock()
	ts, exists := taskStatuses[taskID]
	taskStatusesMu.RUnlock()

	if !exists {
		return
	}

	logger := log.WithPrefix("webhook")

	payload := TaskStatusResponse{
		TaskID:      ts.ID,
		Status:      status,
		Title:       ts.Title,
		CreatedAt:   ts.CreatedAt,
		Error:       errorMsg,
		Downloaded:  ts.Downloaded.Load(),
		Total:       ts.Total.Load(),
		ProgressPct: math.Float64frombits(atomic.LoadUint64((*uint64)(&ts.ProgressPct))),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf("Failed to marshal webhook payload: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.API.WebhookURL, bytes.NewReader(body))
	if err != nil {
		logger.Errorf("Failed to create webhook request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.API.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.API.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Errorf("Failed to send webhook: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("Webhook returned error status %d, failed to read response body: %v", resp.StatusCode, err)
		} else {
			logger.Errorf("Webhook returned error status %d: %s", resp.StatusCode, string(body))
		}
	}
}
