package types

import (
	"gorm.io/gorm"
)

type ReceivedFile struct {
	gorm.Model
	Processing     bool
	ChatID         int64
	MessageID      int
	ReplyMessageID int
	FileName       string
}

type User struct {
	gorm.Model
	UserID         int64 `gorm:"uniqueIndex"`
	Silent         bool
	DefaultStorage string
}
