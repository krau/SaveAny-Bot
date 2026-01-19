package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	tftask "github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/pkg/queue"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

// Request/Response types
type CreateTaskRequest struct {
	TelegramURL  string `json:"telegram_url"`
	StorageName  string `json:"storage_name,omitempty"`
	DirPath      string `json:"dir_path,omitempty"`
	UserID       int64  `json:"user_id"`
}

type CreateTaskResponse struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

type TaskStatusResponse struct {
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"` // queued, running, completed, failed, canceled
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Error     string    `json:"error,omitempty"`
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
	ID        string
	Status    string
	Title     string
	CreatedAt time.Time
	Error     string
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
	if req.UserID == 0 {
		respondError(w, "user_id is required", http.StatusBadRequest)
		return
	}

	logger := log.FromContext(r.Context()).WithPrefix("api")

	// Get storage
	var stor storage.Storage
	var err error
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

	// Create TGFile from message media
	tgFile, err := tfile.FromMediaMessage(media, botCtx.Raw, msg)
	if err != nil {
		logger.Errorf("Failed to create TGFile: %v", err)
		respondError(w, "invalid message format", http.StatusBadRequest)
		return
	}

	// Create task
	dirPath := req.DirPath
	if dirPath == "" {
		dirPath = "/"
	}

	storagePath := stor.JoinStoragePath(path.Join(dirPath, tgFile.Name()))
	taskID := xid.New().String()

	// Create context with bot extension
	injectCtx := tgutil.ExtWithContext(r.Context(), botCtx)

	task, err := tftask.NewTGFileTask(taskID, injectCtx, tgFile, stor, storagePath, &apiProgressTracker{
		taskID: taskID,
	})
	if err != nil {
		logger.Errorf("Failed to create task: %v", err)
		respondError(w, fmt.Sprintf("failed to create task: %v", err), http.StatusInternalServerError)
		return
	}

	// Track task status
	trackTask(taskID, task.Title(), "queued")

	// Add task to queue
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("Failed to add task: %v", err)
		updateTaskStatus(taskID, "failed", err.Error())
		respondError(w, fmt.Sprintf("failed to add task: %v", err), http.StatusInternalServerError)
		return
	}

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateTaskResponse{
		TaskID:  taskID,
		Message: "task created successfully",
	})
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
		TaskID:    status.ID,
		Status:    status.Status,
		Title:     status.Title,
		CreatedAt: status.CreatedAt,
		Error:     status.Error,
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
		respondError(w, fmt.Sprintf("failed to cancel task: %v", err), http.StatusInternalServerError)
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
	// No-op for API tasks
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

	payload := TaskStatusResponse{
		TaskID:    ts.ID,
		Status:    status,
		Title:     ts.Title,
		CreatedAt: ts.CreatedAt,
		Error:     errorMsg,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Errorf("Failed to marshal webhook payload: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.API.WebhookURL, bytes.NewReader(body))
	if err != nil {
		log.Errorf("Failed to create webhook request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.API.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.API.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("Failed to send webhook: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Errorf("Webhook returned error status %d: %s", resp.StatusCode, string(body))
	}
}
