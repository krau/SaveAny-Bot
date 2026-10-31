package database

import (
	"context"
	"time"
)

func CreateTaskRecord(ctx context.Context, record *TaskRecord) error {
	return db.WithContext(ctx).Create(record).Error
}

func UpdateTaskRecord(ctx context.Context, taskID string, fields map[string]any) error {
	return db.WithContext(ctx).Model(&TaskRecord{}).Where("task_id = ?", taskID).Updates(fields).Error
}

func GetTaskRecord(ctx context.Context, taskID string) (*TaskRecord, error) {
	var record TaskRecord
	if err := db.WithContext(ctx).Where("task_id = ?", taskID).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func ListTaskRecords(ctx context.Context, filter TaskFilter) ([]TaskRecord, int64, error) {
	query := db.WithContext(ctx).Model(&TaskRecord{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var records []TaskRecord
	err := query.Order("created_at DESC").Limit(limit).Offset(filter.Offset).Find(&records).Error
	return records, total, err
}

func DeleteTaskRecord(ctx context.Context, taskID string) error {
	return db.WithContext(ctx).Unscoped().Where("task_id = ?", taskID).Delete(&TaskRecord{}).Error
}

func MarkInterruptedTaskRecords(ctx context.Context) error {
	return db.WithContext(ctx).Model(&TaskRecord{}).
		Where("status IN ?", []string{"queued", "running"}).
		Updates(map[string]any{
			"status":      "failed",
			"error":       "interrupted by restart",
			"finished_at": time.Now(),
		}).Error
}
