package dao

func CreateUser(chatID int64) error {
	if _, err := GetUserByChatID(chatID); err == nil {
		return nil
	}
	return db.Create(&User{ChatID: chatID}).Error
}

func GetAllUsers() ([]User, error) {
	var users []User
	err := db.Preload("Dirs").
		Preload("Rules").
		Find(&users).Error
	return users, err
}

func GetUserByChatID(chatID int64) (*User, error) {
	var user User
	err := db.
		Preload("Dirs").
		Preload("Rules").
		Where("chat_id = ?", chatID).First(&user).Error
	return &user, err
}

func UpdateUser(user *User) error {
	return db.Save(user).Error
}

func DeleteUser(user *User) error {
	return db.Unscoped().Select("Dirs", "Rules").Delete(user).Error
}
