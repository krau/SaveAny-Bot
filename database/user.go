package database

import "context"

// CreateUser creates a new user in the database (SQLite or Redis)
func CreateUser(ctx context.Context, chatID int64) error {
	if useRedis {
		return redisCreateUser(ctx, chatID)
	}
	
	// SQLite implementation (original)
	if _, err := GetUserByChatID(ctx, chatID); err == nil {
		return nil
	}
	return db.Create(&User{ChatID: chatID}).Error
}

// GetAllUsers retrieves all users from the database (SQLite or Redis)
func GetAllUsers(ctx context.Context) ([]User, error) {
	if useRedis {
		return redisGetAllUsers(ctx)
	}
	
	// SQLite implementation (original)
	var users []User
	err := db.Preload("Dirs").
		WithContext(ctx).
		Preload("Rules").
		Find(&users).Error
	return users, err
}

// GetUserByChatID retrieves a user by chat ID from the database (SQLite or Redis)
func GetUserByChatID(ctx context.Context, chatID int64) (*User, error) {
	if useRedis {
		return redisGetUserByChatID(ctx, chatID)
	}
	
	// SQLite implementation (original)
	var user User
	err := db.
		Preload("Dirs").
		WithContext(ctx).
		Preload("Rules").
		Where("chat_id = ?", chatID).First(&user).Error
	return &user, err
}

// UpdateUser updates a user in the database (SQLite or Redis)
func UpdateUser(ctx context.Context, user *User) error {
	if useRedis {
		return redisUpdateUser(ctx, user)
	}
	
	// SQLite implementation (original)
	if _, err := GetUserByChatID(ctx, user.ChatID); err != nil {
		return err
	}
	return db.WithContext(ctx).Save(user).Error
}

// DeleteUser deletes a user from the database (SQLite or Redis)
func DeleteUser(ctx context.Context, user *User) error {
	if useRedis {
		return redisDeleteUser(ctx, user)
	}
	
	// SQLite implementation (original)
	return db.WithContext(ctx).Unscoped().Select("Dirs", "Rules").Delete(user).Error
}
