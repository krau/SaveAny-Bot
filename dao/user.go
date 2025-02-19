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

func GetAllUsers() ([]types.User, error) {
	var users []types.User
	err := db.Find(&users).Error
	return users, err
}

func GetUserByChatID(chatID int64) (*types.User, error) {
	var user types.User
	err := db.Where("chat_id = ?", chatID).First(&user).Error
	return &user, err
}

func UpdateUser(user *types.User) error {
	return db.Save(user).Error
}

func DeleteUser(user *types.User) error {
	return db.Unscoped().Delete(user).Error
}
