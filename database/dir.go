package database

import (
	"context"
	"fmt"
)

// CreateDirForUser creates a directory for a user in the database (SQLite or Redis)
func CreateDirForUser(ctx context.Context, userID uint, storageName, path string) error {
	if useRedis {
		return redisCreateDirForUser(ctx, userID, storageName, path)
	}
	
	// SQLite implementation (original)
	dir := Dir{
		UserID:      userID,
		StorageName: storageName,
		Path:        path,
	}
	return db.WithContext(ctx).Create(&dir).Error
}

// GetDirByID retrieves a directory by ID from the database (SQLite or Redis)
func GetDirByID(ctx context.Context, id uint) (*Dir, error) {
	if useRedis {
		// For Redis, we need to find the userID first since we don't have a global dir index
		// This is a limitation of our Redis design - we'll need to search through user dirs
		// For now, return an error as this function would be inefficient in Redis
		// In a production system, you might want to maintain a global dir index
		return nil, fmt.Errorf("GetDirByID not efficiently supported with Redis - use GetUserDirs instead")
	}
	
	// SQLite implementation (original)
	dir := &Dir{}
	err := db.WithContext(ctx).First(dir, id).Error
	if err != nil {
		return nil, err
	}
	return dir, err
}

// GetUserDirs retrieves directories for a user from the database (SQLite or Redis)
func GetUserDirs(ctx context.Context, userID uint) ([]Dir, error) {
	if useRedis {
		return redisGetUserDirs(ctx, userID)
	}
	
	// SQLite implementation (original)
	var dirs []Dir
	err := db.WithContext(ctx).Where("user_id = ?", userID).Find(&dirs).Error
	return dirs, err
}

// GetUserDirsByChatID retrieves directories for a user by chat ID from the database (SQLite or Redis)
func GetUserDirsByChatID(ctx context.Context, chatID int64) ([]Dir, error) {
	if useRedis {
		return redisGetUserDirsByChatID(ctx, chatID)
	}
	
	// SQLite implementation (original)
	user, err := GetUserByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return GetUserDirs(ctx, user.ID)
}

// GetDirsByUserIDAndStorageName retrieves directories by user ID and storage name from the database (SQLite or Redis)
func GetDirsByUserIDAndStorageName(ctx context.Context, userID uint, storageName string) ([]Dir, error) {
	if useRedis {
		return redisGetDirsByUserIDAndStorageName(ctx, userID, storageName)
	}
	
	// SQLite implementation (original)
	var dirs []Dir
	err := db.WithContext(ctx).Where("user_id = ? AND storage_name = ?", userID, storageName).Find(&dirs).Error
	return dirs, err
}

// GetDirsByUserChatIDAndStorageName retrieves directories by user chat ID and storage name from the database (SQLite or Redis)
func GetDirsByUserChatIDAndStorageName(ctx context.Context, chatID int64, storageName string) ([]Dir, error) {
	if useRedis {
		return redisGetDirsByUserChatIDAndStorageName(ctx, chatID, storageName)
	}
	
	// SQLite implementation (original)
	user, err := GetUserByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return GetDirsByUserIDAndStorageName(ctx, user.ID, storageName)
}

// DeleteDirForUser deletes a directory for a user from the database (SQLite or Redis)
func DeleteDirForUser(ctx context.Context, userID uint, storageName, path string) error {
	if useRedis {
		return redisDeleteDirForUser(ctx, userID, storageName, path)
	}
	
	// SQLite implementation (original)
	return db.WithContext(ctx).Unscoped().Where("user_id = ? AND storage_name = ? AND path = ?", userID, storageName, path).Delete(&Dir{}).Error
}

// DeleteDirByID deletes a directory by ID from the database (SQLite or Redis)
func DeleteDirByID(ctx context.Context, id uint) error {
	if useRedis {
		return redisDeleteDirByID(ctx, id)
	}
	
	// SQLite implementation (original)
	return db.WithContext(ctx).Unscoped().Delete(&Dir{}, id).Error
}
