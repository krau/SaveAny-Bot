package database

import "context"

func CreateDirForUser(ctx context.Context, userID, storageID uint, path string) error {
	dir := Dir{
		UserID:    userID,
		StorageID: storageID,
		Path:      path,
	}
	return db.WithContext(ctx).Create(&dir).Error
}

func GetDirByID(ctx context.Context, id uint) (*Dir, error) {
	dir := &Dir{}
	err := db.WithContext(ctx).First(dir, id).Error
	if err != nil {
		return nil, err
	}
	return dir, err
}

func GetUserDirs(ctx context.Context, userID uint) ([]Dir, error) {
	var dirs []Dir
	err := db.WithContext(ctx).Where("user_id = ?", userID).Find(&dirs).Error
	return dirs, err
}

func GetUserDirsByChatID(ctx context.Context, chatID int64) ([]Dir, error) {
	user, err := GetUserByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return GetUserDirs(ctx, user.ID)
}

func GetDirsByUserIDAndStorageID(ctx context.Context, userID, storageID uint) ([]Dir, error) {
	var dirs []Dir
	err := db.WithContext(ctx).Where("user_id = ? AND storage_id = ?", userID, storageID).Find(&dirs).Error
	return dirs, err
}

func GetDirsByUserChatIDAndStorageID(ctx context.Context, chatID int64, storageID uint) ([]Dir, error) {
	user, err := GetUserByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return GetDirsByUserIDAndStorageID(ctx, user.ID, storageID)
}

func DeleteDirForUser(ctx context.Context, userID, storageID uint, path string) error {
	return db.WithContext(ctx).Unscoped().Where("user_id = ? AND storage_id = ? AND path = ?", userID, storageID, path).Delete(&Dir{}).Error
}

func DeleteDirByID(ctx context.Context, id uint) error {
	return db.WithContext(ctx).Unscoped().Delete(&Dir{}, id).Error
}
