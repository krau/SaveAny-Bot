package dao

import (
	"github.com/krau/SaveAny-Bot/model"
)

func CreateUser(userID int64) error {
	if _, err := GetUserByUserID(userID); err == nil {
		return nil
	}
	return db.Create(&model.User{UserID: userID}).Error
}

func GetUserByUserID(userID int64) (*model.User, error) {
	var user model.User
	err := db.Where("user_id = ?", userID).First(&user).Error
	return &user, err
}

func UpdateUser(user *model.User) error {
	return db.Save(user).Error
}
