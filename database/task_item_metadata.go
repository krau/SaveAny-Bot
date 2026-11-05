package database

import (
	"context"

	"gorm.io/gorm"
)

// ReplaceTaskItemMetadata stores normalized metadata for a task, replacing any
// rows left by an earlier execution of the same task.
func ReplaceTaskItemMetadata(ctx context.Context, taskID string, records []TaskItemMetadata) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("task_id = ?", taskID).Delete(&TaskItemMetadata{}).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		for i := range records {
			records[i].TaskID = taskID
		}
		return tx.CreateInBatches(records, 100).Error
	})
}

func ListTaskItemMetadata(ctx context.Context, taskID string) ([]TaskItemMetadata, error) {
	var records []TaskItemMetadata
	err := db.WithContext(ctx).Where("task_id = ?", taskID).Order("item_index ASC").Find(&records).Error
	return records, err
}

func CountTaskMetadata(ctx context.Context, taskIDs []string) (map[string]int, error) {
	counts := make(map[string]int, len(taskIDs))
	if len(taskIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		TaskID string
		Count  int
	}
	err := db.WithContext(ctx).Model(&TaskItemMetadata{}).
		Select("task_id, COUNT(*) AS count").
		Where("task_id IN ?", taskIDs).
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
