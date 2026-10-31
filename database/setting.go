package database

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetSetting(ctx context.Context, key string) (string, bool, error) {
	var setting Setting
	err := db.WithContext(ctx).Where("key = ?", key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", false, nil
		}
		return "", false, err
	}
	return setting.Value, true, nil
}

func SetSetting(ctx context.Context, key, value string) error {
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&Setting{Key: key, Value: value}).Error
}

func SetSettingsTx(ctx context.Context, values map[string]string) error {
	return Transaction(ctx, func(tx *gorm.DB) error {
		for key, value := range values {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
			}).Create(&Setting{Key: key, Value: value}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ListSettings(ctx context.Context) (map[string]string, error) {
	var settings []Setting
	if err := db.WithContext(ctx).Find(&settings).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(settings))
	for _, s := range settings {
		out[s.Key] = s.Value
	}
	return out, nil
}
