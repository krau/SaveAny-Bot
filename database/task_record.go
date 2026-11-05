package database

import (
	"context"

	"gorm.io/gorm"

	"github.com/krau/SaveAny-Bot/pkg/enums/taskstatus"
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

// ListQueuedTaskRecords returns every task that is queued for execution,
// including tasks requeued after an interrupted run, oldest first.
func ListQueuedTaskRecords(ctx context.Context) ([]TaskRecord, error) {
	var records []TaskRecord
	err := db.WithContext(ctx).
		Where("status = ?", taskstatus.Queued.String()).
		Order("created_at ASC").
		Find(&records).Error
	return records, err
}

// HasUnfinishedTasks reports whether any task is queued or running.
func HasUnfinishedTasks(ctx context.Context) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&TaskRecord{}).
		Where("status IN ?", []string{taskstatus.Queued.String(), taskstatus.Running.String()}).
		Count(&count).Error
	return count > 0, err
}

func DeleteTaskRecord(ctx context.Context, taskID string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("task_id = ?", taskID).Delete(&TaskItemMetadata{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("task_id = ?", taskID).Delete(&TaskItemRecord{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Where("task_id = ?", taskID).Delete(&TaskRecord{}).Error
	})
}

// ReplaceTaskItems stores the final outcome of every file in a task,
// replacing any previous rows for the same task.
func ReplaceTaskItems(ctx context.Context, taskID string, items []TaskItemRecord) error {
	if len(items) == 0 {
		return nil
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("task_id = ?", taskID).Delete(&TaskItemRecord{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TaskID = taskID
		}
		return tx.CreateInBatches(items, 100).Error
	})
}

func CountFailedTaskItems(ctx context.Context, taskIDs []string) (map[string]int, error) {
	counts := make(map[string]int, len(taskIDs))
	if len(taskIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		TaskID string
		Count  int
	}
	err := db.WithContext(ctx).Model(&TaskItemRecord{}).
		Select("task_id, COUNT(*) AS count").
		Where("task_id IN ? AND status = ?", taskIDs, taskstatus.Failed.String()).
		Group("task_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.TaskID] = row.Count
	}
	return counts, nil
}

func ListTaskItems(ctx context.Context, taskID string) ([]TaskItemRecord, error) {
	var items []TaskItemRecord
	if err := db.WithContext(ctx).Where("task_id = ?", taskID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListFailedTaskItems returns failed items for the given task IDs, capped per
// task so a large batch cannot blow up a list response.
func ListFailedTaskItems(ctx context.Context, taskIDs []string, perTask int) (map[string][]TaskItemRecord, error) {
	out := make(map[string][]TaskItemRecord, len(taskIDs))
	if len(taskIDs) == 0 {
		return out, nil
	}
	if perTask <= 0 {
		perTask = 20
	}
	// Cap rows per task in SQL so a task with a large number of failures
	// cannot make every list request load all of them.
	const query = `
SELECT * FROM (
	SELECT task_items.*, ROW_NUMBER() OVER (PARTITION BY task_id ORDER BY id) AS row_num
	FROM task_items
	WHERE task_id IN ? AND status = ?
) WHERE row_num <= ?`
	var items []TaskItemRecord
	if err := db.WithContext(ctx).Raw(query, taskIDs, taskstatus.Failed.String(), perTask).Scan(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		group := out[item.TaskID]
		if len(group) >= perTask {
			continue
		}
		out[item.TaskID] = append(group, item)
	}
	return out, nil
}

// MarkInterruptedTaskRecords requeues tasks that were running when the
// process stopped so they can be rebuilt from their persisted parameters.
func MarkInterruptedTaskRecords(ctx context.Context) error {
	return db.WithContext(ctx).Model(&TaskRecord{}).
		Where("status = ?", taskstatus.Running.String()).
		Updates(map[string]any{
			"status":      taskstatus.Queued.String(),
			"error":       "",
			"started_at":  nil,
			"finished_at": nil,
		}).Error
}
