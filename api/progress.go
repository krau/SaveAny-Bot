package api

import (
	"sync"
	"sync/atomic"
	"time"
)

// TaskProgressInfo 存储任务的进度信息
type TaskProgressInfo struct {
	TaskID          string
	Type            string
	Status          TaskStatus
	Title           string
	TotalBytes      int64
	DownloadedBytes int64
	TotalFiles      int
	DownloadedFiles int
	Storage         string
	Path            string
	Error           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Webhook         string
}

// progressStore 存储所有 API 任务的进度信息
type progressStore struct {
	mu    sync.RWMutex
	tasks map[string]*TaskProgressInfo
}

var store = &progressStore{
	tasks: make(map[string]*TaskProgressInfo),
}

// RegisterTask 注册一个新的 API 任务
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

// GetTask 获取任务进度信息
func GetTask(taskID string) (*TaskProgressInfo, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	info, ok := store.tasks[taskID]
	return info, ok
}

// GetAllTasks 获取所有任务
func GetAllTasks() []*TaskProgressInfo {
	store.mu.RLock()
	defer store.mu.RUnlock()

	tasks := make([]*TaskProgressInfo, 0, len(store.tasks))
	for _, info := range store.tasks {
		tasks = append(tasks, info)
	}
	return tasks
}

// DeleteTask 删除任务记录
func DeleteTask(taskID string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.tasks, taskID)
}

// UpdateStatus 更新任务状态
func (t *TaskProgressInfo) UpdateStatus(status TaskStatus) {
	t.Status = status
	t.UpdatedAt = time.Now()
}

// SetError 设置错误信息
func (t *TaskProgressInfo) SetError(err string) {
	t.Error = err
	t.Status = TaskStatusFailed
	t.UpdatedAt = time.Now()
}

// ProgressTracker 用于 API 任务的进度追踪
type ProgressTracker struct {
	info *TaskProgressInfo
}

// NewProgressTracker 创建新的进度追踪器
func NewProgressTracker(taskID, taskType, storage, path, title, webhook string) *ProgressTracker {
	info := RegisterTask(taskID, taskType, storage, path, title, webhook)
	return &ProgressTracker{info: info}
}

// OnStart 任务开始
func (p *ProgressTracker) OnStart(totalBytes int64, totalFiles int) {
	p.info.Status = TaskStatusRunning
	p.info.TotalBytes = totalBytes
	p.info.TotalFiles = totalFiles
	p.info.UpdatedAt = time.Now()
}

// OnProgress 进度更新
func (p *ProgressTracker) OnProgress(downloadedBytes int64, downloadedFiles int) {
	atomic.StoreInt64(&p.info.DownloadedBytes, downloadedBytes)
	p.info.DownloadedFiles = downloadedFiles
	p.info.UpdatedAt = time.Now()
}

// OnDone 任务完成
func (p *ProgressTracker) OnDone(err error) {
	if err != nil {
		p.info.Status = TaskStatusFailed
		p.info.Error = err.Error()
	} else {
		p.info.Status = TaskStatusCompleted
	}
	p.info.UpdatedAt = time.Now()
}

// GetInfo 获取任务信息
func (p *ProgressTracker) GetInfo() *TaskProgressInfo {
	return p.info
}

// UpdateProgressBytes 更新下载字节数
func (p *ProgressTracker) UpdateProgressBytes(bytes int64) {
	atomic.StoreInt64(&p.info.DownloadedBytes, bytes)
	p.info.UpdatedAt = time.Now()
}

// UpdateProgressFiles 更新下载文件数
func (p *ProgressTracker) UpdateProgressFiles(files int) {
	p.info.DownloadedFiles = files
	p.info.UpdatedAt = time.Now()
}
