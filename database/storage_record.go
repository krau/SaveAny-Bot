package database

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"gorm.io/gorm"
)

func CreateStorageRecord(ctx context.Context, record *StorageRecord) error {
	return db.WithContext(ctx).Create(record).Error
}

func ListStorageRecords(ctx context.Context) ([]StorageRecord, error) {
	var records []StorageRecord
	err := db.WithContext(ctx).Order("name").Find(&records).Error
	return records, err
}

func GetStorageRecord(ctx context.Context, id uint) (*StorageRecord, error) {
	var record StorageRecord
	if err := db.WithContext(ctx).First(&record, id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func UpdateStorageRecord(ctx context.Context, record *StorageRecord) error {
	return db.WithContext(ctx).Save(record).Error
}

func DeleteStorageRecord(ctx context.Context, id uint) error {
	return db.WithContext(ctx).Unscoped().Delete(&StorageRecord{}, id).Error
}

// StorageReferences returns how many entities still reference the storage name.
func StorageReferences(ctx context.Context, name string) (users, dirs, rules int64, err error) {
	var allUsers []User
	if err = db.WithContext(ctx).Find(&allUsers).Error; err != nil {
		return 0, 0, 0, err
	}
	for _, u := range allUsers {
		if u.DefaultStorage == name {
			users++
			continue
		}
		var names []string
		if u.StoragesJSON != "" {
			_ = json.Unmarshal([]byte(u.StoragesJSON), &names)
		}
		if slices.Contains(names, name) {
			users++
		}
	}
	if err = db.WithContext(ctx).Model(&Dir{}).Where("storage_name = ?", name).Count(&dirs).Error; err != nil {
		return
	}
	if err = db.WithContext(ctx).Model(&Rule{}).Where("storage_name = ?", name).Count(&rules).Error; err != nil {
		return
	}
	return users, dirs, rules, nil
}

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
