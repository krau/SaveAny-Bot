package dao

func CreateDirForUser(userID uint, storageName, path string) error {
	dir := Dir{
		UserID:      userID,
		StorageName: storageName,
		Path:        path,
	}
	return db.Create(&dir).Error
}

func GetDirByID(id uint) (*Dir, error) {
	dir := &Dir{}
	err := db.First(dir, id).Error
	if err != nil {
		return nil, err
	}
	return dir, err
}

func GetUserDirs(userID uint) ([]Dir, error) {
	var dirs []Dir
	err := db.Where("user_id = ?", userID).Find(&dirs).Error
	return dirs, err
}

func GetUserDirsByChatID(chatID int64) ([]Dir, error) {
	user, err := GetUserByChatID(chatID)
	if err != nil {
		return nil, err
	}
	return GetUserDirs(user.ID)
}

func GetDirsByUserIDAndStorageName(userID uint, storageName string) ([]Dir, error) {
	var dirs []Dir
	err := db.Where("user_id = ? AND storage_name = ?", userID, storageName).Find(&dirs).Error
	return dirs, err
}

func DeleteDirForUser(userID uint, storageName, path string) error {
	return db.Unscoped().Where("user_id = ? AND storage_name = ? AND path = ?", userID, storageName, path).Delete(&Dir{}).Error
}

func DeleteDirByID(id uint) error {
	return db.Unscoped().Delete(&Dir{}, id).Error
}