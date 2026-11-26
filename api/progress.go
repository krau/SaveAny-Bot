package api

import (
	"context"
	"sync"
	"time"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
)

// TaskProgressInfo stores live progress for a task. All fields are guarded by
// mu. It implements taskevent.Sink so the task layer can update it without
// knowing about the API.
type TaskProgressInfo struct {
	mu              sync.Mutex
	TaskID          string
	Type            string
	Status          TaskStatus
	Title           string
	TotalBytes      int64
	DownloadedBytes int64
	TotalFiles      int
	DownloadedFiles int
	StorageID       uint
	Storage         string
	Path            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Webhook         string
	webhookNotified bool
}

// progressStore holds all tracked tasks. Entries are removed a fixed duration
// after they reach a terminal state to bound memory usage.
type progressStore struct {
	mu        sync.RWMutex
	tasks     map[string]*TaskProgressInfo
	retention time.Duration
}

var store = &progressStore{
	tasks:     make(map[string]*TaskProgressInfo),
	retention: 24 * time.Hour,
}

// TaskProgressSink creates the in-memory observer used by tasks started
// outside the API. Their progress is rendered in the task list but no webhook
// is attached.
func TaskProgressSink(task core.Executable) taskevent.Sink {
	storageName := ""
	storagePath := ""
	if destination, ok := task.(core.DestinationInfo); ok {
		storageName = destination.StorageName()
		storagePath = destination.StoragePath()
	}
	return RegisterTask(task.TaskID(), task.Type().String(), storageName, storagePath, task.Title(), "")
}

func RegisterTask(taskID, taskType, storage, path, title, webhook string) *TaskProgressInfo {
	info := &TaskProgressInfo{
		TaskID:    taskID,
		Type:      taskType,
		Status:    TaskStatusQueued,
		Title:     title,
		StorageID: config.C().GetStorageIDByName(storage),
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

func GetTask(taskID string) (*TaskProgressInfo, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	info, ok := store.tasks[taskID]
	return info, ok
}

func GetAllTasks() []*TaskProgressInfo {
	store.mu.RLock()
	defer store.mu.RUnlock()

	tasks := make([]*TaskProgressInfo, 0, len(store.tasks))
	for _, info := range store.tasks {
		tasks = append(tasks, info)
	}
	return tasks
}

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
		terminal := info.Status.IsTerminal()
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

// snapshot returns a point-in-time copy of the fields needed to render a
// response, so callers never touch the mutex directly.
func (t *TaskProgressInfo) snapshot() (total, downloaded int64, totalFiles, downloadedFiles int, updatedAt time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.TotalBytes, t.DownloadedBytes, t.TotalFiles, t.DownloadedFiles, t.UpdatedAt
}

// Emit implements taskevent.Sink. It translates task lifecycle events into
// status/progress updates and fires the webhook on terminal transitions.
func (t *TaskProgressInfo) Emit(e taskevent.Event) {
	t.mu.Lock()
	if t.Status.IsTerminal() {
		t.mu.Unlock()
		return
	}
	switch e.Phase {
	case taskevent.PhaseStart:
		t.Status = TaskStatusRunning
		if e.TotalBytes > 0 {
			t.TotalBytes = e.TotalBytes
		}
		if e.TotalFiles > 0 {
			t.TotalFiles = e.TotalFiles
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
		if e.Path != "" {
			t.Path = e.Path
		}
		switch {
		case e.Cancelled:
			t.Status = TaskStatusCancelled
		case e.Err != nil:
			t.Status = TaskStatusFailed
		case e.Partial:
			t.Status = TaskStatusPartial
		default:
			t.Status = TaskStatusCompleted
		}
	}
	t.UpdatedAt = time.Now()
	notify := t.Webhook != "" && !t.webhookNotified && t.Status.IsTerminal() && t.Status != TaskStatusCancelled
	status := t.Status
	path := t.Path
	if notify {
		t.webhookNotified = true
	}
	t.mu.Unlock()

	if notify {
		payload := CreateWebhookPayload(t.TaskID, t.Type, status, t.StorageID, t.Storage, path, e.Err)
		SendWebhook(context.Background(), payload)
	}
}
