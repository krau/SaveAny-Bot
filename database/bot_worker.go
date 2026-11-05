package database

import "context"

func CreateBotWorker(ctx context.Context, worker *BotWorker) error {
	return db.WithContext(ctx).Create(worker).Error
}

func ListBotWorkers(ctx context.Context) ([]BotWorker, error) {
	var workers []BotWorker
	if err := db.WithContext(ctx).Order("id ASC").Find(&workers).Error; err != nil {
		return nil, err
	}
	return workers, nil
}

func GetBotWorker(ctx context.Context, id uint) (*BotWorker, error) {
	var worker BotWorker
	if err := db.WithContext(ctx).First(&worker, id).Error; err != nil {
		return nil, err
	}
	return &worker, nil
}

func UpdateBotWorker(ctx context.Context, id uint, fields map[string]any) error {
	return db.WithContext(ctx).Model(&BotWorker{}).Where("id = ?", id).Updates(fields).Error
}

func DeleteBotWorker(ctx context.Context, id uint) error {
	return db.WithContext(ctx).Unscoped().Delete(&BotWorker{}, id).Error
}
