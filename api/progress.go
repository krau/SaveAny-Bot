package api

import (
	"sync"
	"time"

	"github.com/krau/SaveAny-Bot/pkg/taskevent"
)

// TaskProgressInfo stores the progress of an API-submitted task. All fields are
// guarded by mu. It implements taskevent.Sink so the task layer can update it
// without knowing about the API.
type TaskProgressInfo struct {
	mu               sync.Mutex
	TaskID           string
	Type             string
	Status           TaskStatus
	Title            string
	TotalBytes       int64
	DownloadedBytes  int64
	TotalFiles       int
	DownloadedFiles  int
	Storage          string
	Path             string
	Error            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	StartedAt        time.Time
	Webhook          string
	webhookNotified  bool
}

// progressStore holds all API tasks. Entries are removed a fixed duration after
// they reach a terminal state to bound memory usage.
type progressStore struct {
	mu        sync.RWMutex
	tasks     map[string]*TaskProgressInfo
	retention time.Duration
}

var store = &progressStore{
	tasks:     make(map[string]*TaskProgressInfo),
	retention: 24 * time.Hour,
}

// RegisterTask registers a new API task and returns its progress info.
func RegisterTask(taskID, taskType, storage, path, title, webhook string) *TaskProgressInfo {
	info := &TaskProgressInfo{
		TaskID:    taskID,
		Type:      taskType,
		Status:    TaskStatusQueued,
		Title:     title,
		Storage:   storage,
		Path:      path,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Webhook:   webhook,
	}

	store.mu.Lock()
	store.tasks[taskID] = info
	store.mu.Unlock()

	return info
}

// GetTask returns the progress info for a task.
func GetTask(taskID string) (*TaskProgressInfo, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	info, ok := store.tasks[taskID]
	return info, ok
}

// GetAllTasks returns all tracked tasks.
func GetAllTasks() []*TaskProgressInfo {
	store.mu.RLock()
	defer store.mu.RUnlock()

	tasks := make([]*TaskProgressInfo, 0, len(store.tasks))
	for _, info := range store.tasks {
		tasks = append(tasks, info)
	}
	return tasks
}

// DeleteTask removes a task record.
func DeleteTask(taskID string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.tasks, taskID)
}

// CleanupExpired removes tasks that reached a terminal state more than the
// store's retention duration ago. It is safe to call periodically.
func CleanupExpired() {
	now := time.Now()
	store.mu.Lock()
	defer store.mu.Unlock()
	for id, info := range store.tasks {
		info.mu.Lock()
		terminal := info.Status == TaskStatusCompleted || info.Status == TaskStatusFailed || info.Status == TaskStatusCancelled
		stale := terminal && now.Sub(info.UpdatedAt) > store.retention
		info.mu.Unlock()
		if stale {
			delete(store.tasks, id)
		}
	}
}

// StartCleanupLoop runs CleanupExpired on a fixed interval until ctx is done.
// It should be started once during API server initialization.
func StartCleanupLoop(ctx interface{ Done() <-chan struct{} }) {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				CleanupExpired()
			}
		}
	}()
}

// UpdateStatus sets the task status.
func (t *TaskProgressInfo) UpdateStatus(status TaskStatus) {
	t.mu.Lock()
	t.Status = status
	t.UpdatedAt = time.Now()
	if status == TaskStatusRunning && t.StartedAt.IsZero() {
		t.StartedAt = t.UpdatedAt
	}
	t.mu.Unlock()
}

// SetError marks the task failed with an error message.
func (t *TaskProgressInfo) SetError(err string) {
	t.mu.Lock()
	t.Error = err
	t.Status = TaskStatusFailed
	t.UpdatedAt = time.Now()
	t.mu.Unlock()
}

// snapshot returns a point-in-time copy of the fields needed to render a
// response, so callers never touch the mutex directly.
func (t *TaskProgressInfo) snapshot() (status TaskStatus, total, downloaded int64, totalFiles, downloadedFiles int, startedAt time.Time, err string, updatedAt time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Status, t.TotalBytes, t.DownloadedBytes, t.TotalFiles, t.DownloadedFiles, t.StartedAt, t.Error, t.UpdatedAt
}

// Emit implements taskevent.Sink. It translates task lifecycle events into
// status/progress updates and fires the webhook on terminal transitions.
func (t *TaskProgressInfo) Emit(e taskevent.Event) {
	t.mu.Lock()
	switch e.Phase {
	case taskevent.PhaseStart:
		t.Status = TaskStatusRunning
		if t.StartedAt.IsZero() {
			t.StartedAt = time.Now()
		}
		if e.TotalBytes > 0 {
			t.TotalBytes = e.TotalBytes
		}
	case taskevent.PhaseProgress:
		t.Status = TaskStatusRunning
		if e.TotalBytes > 0 {
			t.TotalBytes = e.TotalBytes
		}
		t.DownloadedBytes = e.DownloadedBytes
		if e.TotalFiles > 0 {
			t.TotalFiles = e.TotalFiles
		}
		if e.DownloadedFiles > 0 {
			t.DownloadedFiles = e.DownloadedFiles
		}
	case taskevent.PhaseDone:
		if e.Err != nil {
			t.Status = TaskStatusFailed
			t.Error = e.Err.Error()
		} else {
			t.Status = TaskStatusCompleted
		}
	}
	t.UpdatedAt = time.Now()
	notify := t.Webhook != "" && !t.webhookNotified && (t.Status == TaskStatusCompleted || t.Status == TaskStatusFailed)
	if notify {
		t.webhookNotified = true
	}
	t.mu.Unlock()

	if notify {
		payload := CreateWebhookPayload(t.TaskID, t.Type, t.Status, t.Storage, t.Path, e.Err)
		SendWebhook(nil, payload)
	}
}

// ProgressTracker is retained for compatibility but is no longer the primary
// progress path; taskevent drives updates now. These methods are safe no-ops
// when called on a nil receiver.
type ProgressTracker struct{}

func NewProgressTracker(taskID, taskType, storage, path, title, webhook string) *ProgressTracker {
	return &ProgressTracker{}
}

func (p *ProgressTracker) OnStart(totalBytes int64, totalFiles int)    {}
func (p *ProgressTracker) OnProgress(downloadedBytes int64, downloadedFiles int) {}
func (p *ProgressTracker) OnDone(err error)                            {}
func (p *ProgressTracker) GetInfo() *TaskProgressInfo                  { return nil }
func (p *ProgressTracker) UpdateProgressBytes(bytes int64)             {}
func (p *ProgressTracker) UpdateProgressFiles(files int)               {}
