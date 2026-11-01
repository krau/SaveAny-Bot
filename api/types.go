package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/settings"
)

type Deps struct {
	Ctx      context.Context
	Settings *settings.Service
	Telegram TelegramController
	DataDir  string
	DBPath   string
	WebToken string
}

type StorageInput struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Enable bool           `json:"enable"`
	Config map[string]any `json:"config"`
}

type StorageRecord struct {
	ID     uint           `json:"id"`
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Enable bool           `json:"enable"`
	Config map[string]any `json:"config"`
	Error  string         `json:"error,omitempty"`
}

type BotStatus struct {
	State     string    `json:"state"`
	ID        int64     `json:"id,omitempty"`
	Username  string    `json:"username,omitempty"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

type LoginFlow struct {
	FlowID string `json:"flow_id"`
	State  string `json:"state"`
	Error  string `json:"error,omitempty"`
}

type UserbotStatus struct {
	Enabled  bool       `json:"enabled"`
	State    string     `json:"state"`
	ID       int64      `json:"id,omitempty"`
	Username string     `json:"username,omitempty"`
	Phone    string     `json:"phone,omitempty"`
	Error    string     `json:"error,omitempty"`
	Login    *LoginFlow `json:"login,omitempty"`
}

type TelegramStatus struct {
	Bot     BotStatus     `json:"bot"`
	Userbot UserbotStatus `json:"userbot"`
}

type TelegramController interface {
	Status(ctx context.Context) TelegramStatus
	StartBot(ctx context.Context) error
	StopBot(ctx context.Context, force bool) error
	RestartBot(ctx context.Context, force bool) error
	StartUserbotLogin(ctx context.Context, phone string) (flowID string, err error)
	SubmitUserbotCode(ctx context.Context, flowID, code string) error
	SubmitUserbotPassword(ctx context.Context, flowID, password string) error
	CancelUserbotLogin(ctx context.Context, flowID string) error
	LogoutUserbot(ctx context.Context) error
	ApplyTelegram(ctx context.Context) error
	RestartProcess()
}

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
	TaskID     string            `json:"task_id"`
	Type       tasktype.TaskType `json:"type"`
	Status     TaskStatus        `json:"status"`
	Title      string            `json:"title"`
	Progress   *TaskProgress     `json:"progress,omitempty"`
	Storage    string            `json:"storage"`
	Path       string            `json:"path"`
	Source     string            `json:"source,omitempty"`
	Error      string            `json:"error,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	StartedAt  *time.Time        `json:"started_at,omitempty"`
	FinishedAt *time.Time        `json:"finished_at,omitempty"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type TasksListResponse struct {
	Tasks []TaskInfoResponse `json:"tasks"`
	Total int64              `json:"total"`
}

type UserStorageFilter struct {
	Storages  []string `json:"storages"`
	Blacklist bool     `json:"blacklist"`
}

type UserResponse struct {
	ID               uint              `json:"id"`
	ChatID           int64             `json:"chat_id"`
	Enabled          bool              `json:"enabled"`
	Silent           bool              `json:"silent"`
	DefaultStorage   string            `json:"default_storage"`
	DefaultDir       uint              `json:"default_dir"`
	ApplyRule        bool              `json:"apply_rule"`
	FilenameStrategy string            `json:"filename_strategy"`
	FilenameTemplate string            `json:"filename_template"`
	ConflictStrategy string            `json:"conflict_strategy"`
	WatchNotify      bool              `json:"watch_notify"`
	StorageFilter    UserStorageFilter `json:"storage_filter"`
}

type UserPatch struct {
	Enabled          *bool              `json:"enabled,omitempty"`
	Silent           *bool              `json:"silent,omitempty"`
	DefaultStorage   *string            `json:"default_storage,omitempty"`
	DefaultDir       *uint              `json:"default_dir,omitempty"`
	ApplyRule        *bool              `json:"apply_rule,omitempty"`
	FilenameStrategy *string            `json:"filename_strategy,omitempty"`
	FilenameTemplate *string            `json:"filename_template,omitempty"`
	ConflictStrategy *string            `json:"conflict_strategy,omitempty"`
	WatchNotify      *bool              `json:"watch_notify,omitempty"`
	StorageFilter    *UserStorageFilter `json:"storage_filter,omitempty"`
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
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type APIError struct {
	StatusCode int
	ErrorCode  string
	Message    string
}

func (e *APIError) Error() string { return e.Message }

func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, statusCode int, errCode, message string) error {
	return WriteJSON(w, statusCode, ErrorResponse{Error: errCode, Message: message})
}

func WriteFieldErrors(w http.ResponseWriter, statusCode int, message string, fields map[string]string) error {
	return WriteJSON(w, statusCode, ErrorResponse{Error: "validation_failed", Message: message, Fields: fields})
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
