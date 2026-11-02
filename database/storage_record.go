package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

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

// MigrateMinioStorageType converts legacy MinIO storage records to S3 and
// normalizes their endpoint to the scheme-based form used by the S3 backend.
func MigrateMinioStorageType(ctx context.Context) error {
	var records []StorageRecord
	if err := db.WithContext(ctx).Where("type = ?", "minio").Find(&records).Error; err != nil {
		return err
	}
	for _, record := range records {
		config := make(map[string]any)
		if record.ConfigJSON != "" {
			if err := json.Unmarshal([]byte(record.ConfigJSON), &config); err != nil {
				return fmt.Errorf("decode legacy MinIO storage %s: %w", record.Name, err)
			}
		}
		if endpoint, ok := config["endpoint"].(string); ok && endpoint != "" && !strings.Contains(endpoint, "://") {
			scheme := "http"
			if useSSL, _ := config["use_ssl"].(bool); useSSL {
				scheme = "https"
			}
			config["endpoint"] = scheme + "://" + endpoint
		}
		delete(config, "use_ssl")
		raw, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("encode S3 storage %s: %w", record.Name, err)
		}
		if err := db.WithContext(ctx).Model(&StorageRecord{}).
			Where("id = ?", record.ID).
			Updates(map[string]any{"type": "s3", "config": string(raw)}).Error; err != nil {
			return err
		}
	}
	return nil
}

// StorageReferences returns how many entities still reference the storage ID.
func StorageReferences(ctx context.Context, storageID uint) (users, dirs, rules int64, err error) {
	var allUsers []User
	if err = db.WithContext(ctx).Find(&allUsers).Error; err != nil {
		return 0, 0, 0, err
	}
	for _, u := range allUsers {
		if u.DefaultStorageID == storageID {
			users++
			continue
		}
		var ids []uint
		if u.StorageFilterIDsJSON != "" {
			_ = json.Unmarshal([]byte(u.StorageFilterIDsJSON), &ids)
		}
		if slices.Contains(ids, storageID) {
			users++
		}
	}
	if err = db.WithContext(ctx).Model(&Dir{}).Where("storage_id = ?", storageID).Count(&dirs).Error; err != nil {
		return
	}
	if err = db.WithContext(ctx).Model(&Rule{}).Where("storage_id = ?", storageID).Count(&rules).Error; err != nil {
		return
	}
	return users, dirs, rules, nil
}

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
