package dao

import (
	"github.com/krau/SaveAny-Bot/types"
)

func CreateUser(userID int64) error {
	if _, err := GetUserByUserID(userID); err == nil {
		return nil
	}
	return db.Create(&types.User{UserID: userID}).Error
}

func GetUserByUserID(userID int64) (*types.User, error) {
	var user types.User
	err := db.Where("user_id = ?", userID).First(&user).Error
	return &user, err
}

func UpdateUser(user *types.User) error {
	return db.Save(user).Error
}
