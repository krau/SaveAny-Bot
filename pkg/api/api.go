package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	tftask "github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

// TaskStatus represents the status of a download task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskInfo stores information about a task
type TaskInfo struct {
	ID        string     `json:"id"`
	Status    TaskStatus `json:"status"`
	Progress  float64    `json:"progress"` // 0-100
	Error     string     `json:"error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	FileName  string     `json:"filename,omitempty"`
}

// DownloadRequest represents the request body for download endpoint
type DownloadRequest struct {
	// URL is the Telegram message link (e.g., https://t.me/c/12345/678 or https://t.me/username/123)
	URL string `json:"url"`
	// Storage is the name of the storage to save to
	Storage string `json:"storage"`
	// Path is the optional path to save the file to
	Path string `json:"path,omitempty"`
}

// DownloadResponse represents the response for download endpoint
type DownloadResponse struct {
	Success bool     `json:"success"`
	TaskIDs []string `json:"task_ids,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// TaskResponse represents the response for task status endpoint
type TaskResponse struct {
	Success bool      `json:"success"`
	Task    *TaskInfo `json:"task,omitempty"`
	Error   string    `json:"error,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// TaskStore stores task information
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*TaskInfo
}

// NewTaskStore creates a new TaskStore
func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: make(map[string]*TaskInfo),
	}
}

// Set stores a task
func (s *TaskStore) Set(id string, info *TaskInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[id] = info
}

// Get retrieves a task
func (s *TaskStore) Get(id string) (*TaskInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.tasks[id]
	return info, ok
}

// Update updates a task
func (s *TaskStore) Update(id string, status TaskStatus, progress float64, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if info, ok := s.tasks[id]; ok {
		info.Status = status
		info.Progress = progress
		info.Error = errMsg
		info.UpdatedAt = time.Now()
	}
}

// Server represents the API server
type Server struct {
	config      *Config
	store       *TaskStore
	logger      *log.Logger
	ctx         context.Context
	ectx        *ext.Context
	addTaskFn   func(ctx context.Context, task interface{}) error
	parseLinkFn func(ctx *ext.Context, link string) (int64, int, error)
	getMsgFn    func(ctx *ext.Context, chatID int64, msgID int) (*tg.Message, error)
}

// NewServer creates a new API server
func NewServer(ctx context.Context, cfg *Config, store *TaskStore) *Server {
	return &Server{
		config: cfg,
		store:  store,
		logger: log.FromContext(ctx).WithPrefix("api"),
		ctx:    ctx,
	}
}

// SetExtContext sets the bot's ext.Context
func (s *Server) SetExtContext(ectx *ext.Context) {
	s.ectx = ectx
}

// SetAddTaskFunc sets the function to add tasks
func (s *Server) SetAddTaskFunc(fn func(ctx context.Context, task interface{}) error) {
	s.addTaskFn = fn
}

// SetParseLinkFunc sets the function to parse message links
func (s *Server) SetParseLinkFunc(fn func(ctx *ext.Context, link string) (int64, int, error)) {
	s.parseLinkFn = fn
}

// SetGetMessageFunc sets the function to get messages
func (s *Server) SetGetMessageFunc(fn func(ctx *ext.Context, chatID int64, msgID int) (*tg.Message, error)) {
	s.getMsgFn = fn
}

// handleDownload handles the download endpoint
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, &ErrorResponse{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	// Check auth token
	if !s.checkAuth(r) {
		writeJSON(w, http.StatusUnauthorized, &ErrorResponse{
			Success: false,
			Error:   "unauthorized",
		})
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, &ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid request body: %v", err),
		})
		return
	}

	// Validate request
	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, &ErrorResponse{
			Success: false,
			Error:   "url is required",
		})
		return
	}
	if req.Storage == "" {
		writeJSON(w, http.StatusBadRequest, &ErrorResponse{
			Success: false,
			Error:   "storage is required",
		})
		return
	}

	// Check if bot context is available
	if s.ectx == nil {
		writeJSON(w, http.StatusInternalServerError, &ErrorResponse{
			Success: false,
			Error:   "bot context not initialized",
		})
		return
	}

	// Get storage
	stor, err := storage.GetStorageByName(s.ctx, req.Storage)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, &ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("storage not found: %v", err),
		})
		return
	}

	// Get files from URL
	files, err := s.getFilesFromURL(req.URL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, &ErrorResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to get files: %v", err),
		})
		return
	}

	if len(files) == 0 {
		writeJSON(w, http.StatusBadRequest, &ErrorResponse{
			Success: false,
			Error:   "no files found in the message",
		})
		return
	}

	// Create task for each file
	taskIDs := make([]string, 0, len(files))
	for _, file := range files {
		taskID := xid.New().String()
		taskIDs = append(taskIDs, taskID)

		// Store task info
		s.store.Set(taskID, &TaskInfo{
			ID:        taskID,
			Status:    TaskStatusPending,
			Progress:  0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			FileName:  file.Name(),
		})

		// Create storage path
		storagePath := file.Name()
		if req.Path != "" {
			storagePath = path.Join(req.Path, file.Name())
		}

		// Create task
		task, err := tftask.NewTGFileTask(taskID, s.ctx, file, stor, storagePath, nil)
		if err != nil {
			s.store.Update(taskID, TaskStatusFailed, 0, err.Error())
			s.logger.Errorf("Failed to create task: %v", err)
			continue
		}

		// Add task to queue
		if s.addTaskFn != nil {
			if err := s.addTaskFn(s.ctx, task); err != nil {
				s.store.Update(taskID, TaskStatusFailed, 0, err.Error())
				s.logger.Errorf("Failed to add task: %v", err)
				continue
			}
		}

		s.logger.Infof("Created download task %s for file %s", taskID, file.Name())
	}

	writeJSON(w, http.StatusOK, &DownloadResponse{
		Success: true,
		TaskIDs: taskIDs,
	})
}

// handleTaskStatus handles the task status endpoint
func (s *Server) handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, &ErrorResponse{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	// Check auth token
	if !s.checkAuth(r) {
		writeJSON(w, http.StatusUnauthorized, &ErrorResponse{
			Success: false,
			Error:   "unauthorized",
		})
		return
	}

	// Get task ID from URL path
	taskID := r.PathValue("task_id")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, &ErrorResponse{
			Success: false,
			Error:   "task_id is required",
		})
		return
	}

	// Get task info
	info, ok := s.store.Get(taskID)
	if !ok {
		writeJSON(w, http.StatusNotFound, &ErrorResponse{
			Success: false,
			Error:   "task not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, &TaskResponse{
		Success: true,
		Task:    info,
	})
}

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// checkAuth checks if the request is authorized
func (s *Server) checkAuth(r *http.Request) bool {
	if s.config.Token == "" {
		return true // No auth required
	}
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	// Support "Bearer <token>" format
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	return token == s.config.Token
}

// writeJSON writes JSON response
func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// getFilesFromURL gets files from a Telegram message URL
func (s *Server) getFilesFromURL(url string) ([]tfile.TGFileMessage, error) {
	if s.parseLinkFn == nil || s.getMsgFn == nil {
		return nil, fmt.Errorf("parse link or get message function not set")
	}

	// Parse the message link
	chatID, msgID, err := s.parseLinkFn(s.ectx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message link: %w", err)
	}

	// Get the message
	msg, err := s.getMsgFn(s.ectx, chatID, msgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Check if message has media
	media, ok := msg.GetMedia()
	if !ok || media == nil {
		return nil, fmt.Errorf("message has no media")
	}

	// Create file from media
	file, err := tfile.FromMediaMessage(media, s.ectx.Raw, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to create file from media: %w", err)
	}

	return []tfile.TGFileMessage{file}, nil
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.Debugf("API request: %s %s", r.Method, r.URL.Path)

	// Handle CORS for preflight requests
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers for all responses
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	switch {
	case r.URL.Path == "/api/v1/download":
		s.handleDownload(w, r)
	case r.URL.Path == "/health":
		s.handleHealth(w, r)
	default:
		// Handle task status: /api/v1/task/{task_id}
		if len(r.URL.Path) > len("/api/v1/task/") && r.URL.Path[:len("/api/v1/task/")] == "/api/v1/task/" {
			s.handleTaskStatus(w, r)
			return
		}
		writeJSON(w, http.StatusNotFound, &ErrorResponse{
			Success: false,
			Error:   "not found",
		})
	}
}

// Start starts the API server
func (s *Server) Start() error {
	if !s.config.Enable {
		s.logger.Info("API server is disabled")
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Infof("Starting API server on %s", addr)

	go func() {
		if err := http.ListenAndServe(addr, s); err != nil {
			s.logger.Errorf("API server error: %v", err)
		}
	}()

	return nil
}
