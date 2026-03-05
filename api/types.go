package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
)

// TaskStatus 表示任务状态
type TaskStatus string

const (
	TaskStatusQueued    TaskStatus = "queued"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Type    tasktype.TaskType `json:"type"`
	Storage string            `json:"storage"`
	Path    string            `json:"path"`
	Webhook string            `json:"webhook,omitempty"`
	Params  json.RawMessage   `json:"params"`
}

// CreateTaskResponse 创建任务响应
type CreateTaskResponse struct {
	TaskID    string            `json:"task_id"`
	Type      tasktype.TaskType `json:"type"`
	Status    TaskStatus        `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
}

// TaskProgress 任务进度
type TaskProgress struct {
	TotalBytes      int64   `json:"total_bytes,omitempty"`
	DownloadedBytes int64   `json:"downloaded_bytes,omitempty"`
	Percent         float64 `json:"percent,omitempty"`
	SpeedMBPS       float64 `json:"speed_mbps,omitempty"`
}

// TaskInfoResponse 任务信息响应
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

// TasksListResponse 任务列表响应
type TasksListResponse struct {
	Tasks []TaskInfoResponse `json:"tasks"`
	Total int                `json:"total"`
}

// StoragesResponse 存储列表响应
type StoragesResponse struct {
	Storages []StorageInfo `json:"storages"`
}

// StorageInfo 存储信息
type StorageInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// WebhookPayload Webhook 回调负载
type WebhookPayload struct {
	TaskID      string     `json:"task_id"`
	Type        string     `json:"type"`
	Status      TaskStatus `json:"status"`
	Storage     string     `json:"storage"`
	Path        string     `json:"path"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// APIError API 错误
type APIError struct {
	StatusCode int
	ErrorCode  string
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}

// WriteJSON 写入 JSON 响应
func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// WriteError 写入错误响应
func WriteError(w http.ResponseWriter, statusCode int, errCode, message string) error {
	return WriteJSON(w, statusCode, ErrorResponse{
		Error:   errCode,
		Message: message,
	})
}

// Task 参数结构体

// DirectLinksParams directlinks 任务参数
type DirectLinksParams struct {
	URLs []string `json:"urls"`
}

// YTDLPParams ytdlp 任务参数
type YTDLPParams struct {
	URLs  []string `json:"urls"`
	Flags []string `json:"flags,omitempty"`
}

// Aria2Params aria2 任务参数
type Aria2Params struct {
	URLs    []string          `json:"urls"`
	Options map[string]string `json:"options,omitempty"`
}

// ParsedParams parsed 任务参数
type ParsedParams struct {
	URL string `json:"url"`
}

// TransferParams transfer 任务参数
type TransferParams struct {
	SourceStorage string `json:"source_storage"`
	SourcePath    string `json:"source_path"`
	TargetStorage string `json:"target_storage"`
	TargetPath    string `json:"target_path"`
}

// TGFilesParams tgfiles 任务参数
type TGFilesParams struct {
	MessageLinks []string `json:"message_links"`
}

// TPHPicsParams tphpics 任务参数
type TPHPicsParams struct {
	TelegraphURL string `json:"telegraph_url"`
}
