package dao

import (
	"github.com/krau/SaveAny-Bot/types"
)

func CreateUser(chatID int64) error {
	if _, err := GetUserByChatID(chatID); err == nil {
		return nil
	}
	return db.Create(&types.User{ChatID: chatID}).Error
}

// GetUserByUserID gets a user by their telegram user ID
//
// Return with active storages
func GetUserByChatID(chatID int64) (*types.User, error) {
	var user types.User
	err := db.Preload("Storages", "active = ?", true).Where("chat_id = ?", chatID).First(&user).Error
	return &user, err
}

func GetUserWithAllStoragesByChatID(chatID int64) (*types.User, error) {
	var user types.User
	err := db.Preload("Storages").Where("chat_id = ?", chatID).First(&user).Error
	return &user, err
}

func UpdateUser(user *types.User) error {
	return db.Save(user).Error
}
