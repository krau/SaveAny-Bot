package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
)

type TaskStatus string

const (
	TaskStatusQueued    TaskStatus = "queued"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type CreateTaskRequest struct {
	Type    tasktype.TaskType `json:"type"`
	Storage string            `json:"storage"`
	Path    string            `json:"path"`
	Webhook string            `json:"webhook,omitempty"`
	Params  json.RawMessage   `json:"params"`
}

type CreateTaskResponse struct {
	TaskID    string            `json:"task_id"`
	Type      tasktype.TaskType `json:"type"`
	Status    TaskStatus        `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
}

type TaskProgress struct {
	TotalBytes      int64   `json:"total_bytes,omitempty"`
	DownloadedBytes int64   `json:"downloaded_bytes,omitempty"`
	TotalFiles      int     `json:"total_files,omitempty"`
	DownloadedFiles int     `json:"downloaded_files,omitempty"`
	Percent         float64 `json:"percent,omitempty"`
	SpeedMBPS       float64 `json:"speed_mbps,omitempty"`
}

type TaskInfoResponse struct {
	TaskID    string            `json:"task_id"`
	Type      tasktype.TaskType `json:"type"`
	Status    TaskStatus        `json:"status"`
	Title     string            `json:"title"`
	Progress  *TaskProgress     `json:"progress,omitempty"`
	Storage   string            `json:"storage"`
	Path      string            `json:"path"`
	Error     string            `json:"error,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type TasksListResponse struct {
	Tasks []TaskInfoResponse `json:"tasks"`
	Total int                `json:"total"`
}

type StoragesResponse struct {
	Storages []StorageInfo `json:"storages"`
}

type StorageInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type WebhookPayload struct {
	TaskID      string     `json:"task_id"`
	Type        string     `json:"type"`
	Status      TaskStatus `json:"status"`
	Storage     string     `json:"storage"`
	Path        string     `json:"path"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type APIError struct {
	StatusCode int
	ErrorCode  string
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}

func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, statusCode int, errCode, message string) error {
	return WriteJSON(w, statusCode, ErrorResponse{
		Error:   errCode,
		Message: message,
	})
}

type DirectLinksParams struct {
	URLs []string `json:"urls"`
}

type YTDLPParams struct {
	URLs  []string `json:"urls"`
	Flags []string `json:"flags,omitempty"`
}

type Aria2Params struct {
	URLs    []string          `json:"urls"`
	Options map[string]string `json:"options,omitempty"`
}

type ParsedParams struct {
	URL string `json:"url"`
}

type TransferParams struct {
	SourceStorage string `json:"source_storage"`
	SourcePath    string `json:"source_path"`
	TargetStorage string `json:"target_storage"`
	TargetPath    string `json:"target_path"`
}

type TGFilesParams struct {
	MessageLinks []string `json:"message_links"`
}

type TPHPicsParams struct {
	TelegraphURL string `json:"telegraph_url"`
}

type MediaMetadataResponse struct {
	URL             string  `json:"url"`
	Title           string  `json:"title,omitempty"`
	Thumbnail       string  `json:"thumbnail,omitempty"`
	Uploader        string  `json:"uploader,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
}
