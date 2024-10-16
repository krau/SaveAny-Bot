package model

import (
	"gorm.io/gorm"
)

type ReceivedFile struct {
	gorm.Model
	Processing     bool
	FileName       string
	ChatID         int64
	MessageID      int32
	ReplyMessageID int32
}

type User struct {
	gorm.Model
	UserID         int64 `gorm:"uniqueIndex"`
	Silent         bool
	DefaultStorage string
}
