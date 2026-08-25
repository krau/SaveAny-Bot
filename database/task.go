package database

import (
	"context"
	"errors"
	"time"
)

var errNotInitialized = errors.New("database not initialized")

type TaskStatus string

const (
	TaskStatusQueued    TaskStatus = "queued"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task is the persisted record of a queued or running task, used to recover
// unfinished work after a process restart. Completed tasks are deleted on
// finish, so the table only ever holds queued/running rows.
type Task struct {
	ID        string `gorm:"primaryKey;size:64"`
	Type      string `gorm:"size:32;index"`
	Payload   []byte
	Status    string `gorm:"size:16;index"`
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func CreateTask(ctx context.Context, task *Task) error {
	if db == nil {
		return errNotInitialized
	}
	return db.WithContext(ctx).Create(task).Error
}

// UpsertTask inserts the task or replaces the existing row with the same ID.
func UpsertTask(ctx context.Context, task *Task) error {
	if db == nil {
		return errNotInitialized
	}
	return db.WithContext(ctx).Save(task).Error
}

func UpdateTaskStatus(ctx context.Context, id string, status TaskStatus, errMsg string) error {
	if db == nil {
		return errNotInitialized
	}
	return db.WithContext(ctx).Model(&Task{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"error":      errMsg,
			"updated_at": time.Now(),
		}).Error
}

func GetTask(ctx context.Context, id string) (*Task, error) {
	if db == nil {
		return nil, errNotInitialized
	}
	var task Task
	if err := db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateTaskPayload replaces the payload of an existing task row.
func UpdateTaskPayload(ctx context.Context, id string, payload []byte) error {
	if db == nil {
		return errNotInitialized
	}
	return db.WithContext(ctx).Model(&Task{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"payload":    payload,
			"updated_at": time.Now(),
		}).Error
}

// RestoreTaskCreatedAt restores the original creation time after a
// re-enqueue overwrote it.
func RestoreTaskCreatedAt(ctx context.Context, id string, createdAt time.Time) error {
	if db == nil {
		return errNotInitialized
	}
	return db.WithContext(ctx).Model(&Task{}).
		Where("id = ?", id).
		Update("created_at", createdAt).Error
}

// DeleteStaleFailedTasks removes failed rows older than the given age.
func DeleteStaleFailedTasks(ctx context.Context, maxAge time.Duration) error {
	if db == nil {
		return errNotInitialized
	}
	return db.WithContext(ctx).
		Where("status = ? AND updated_at < ?", string(TaskStatusFailed), time.Now().Add(-maxAge)).
		Delete(&Task{}).Error
}

func DeleteTask(ctx context.Context, id string) error {
	if db == nil {
		return errNotInitialized
	}
	return db.WithContext(ctx).Delete(&Task{}, "id = ?", id).Error
}

// GetUnfinishedTasks returns all tasks that were not finished when the
// process stopped, i.e. tasks that must be re-enqueued on startup.
func GetUnfinishedTasks(ctx context.Context) ([]Task, error) {
	if db == nil {
		return nil, errNotInitialized
	}
	var tasks []Task
	err := db.WithContext(ctx).
		Where("status IN ?", []string{string(TaskStatusQueued), string(TaskStatusRunning)}).
		Order("created_at").
		Find(&tasks).Error
	return tasks, err
}

func CountUnfinishedTasks(ctx context.Context) (int64, error) {
	if db == nil {
		return 0, errNotInitialized
	}
	var count int64
	err := db.WithContext(ctx).
		Model(&Task{}).
		Where("status IN ?", []string{string(TaskStatusQueued), string(TaskStatusRunning)}).
		Count(&count).Error
	return count, err
}
