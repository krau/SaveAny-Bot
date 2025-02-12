package types

import (
	"gorm.io/gorm"
)

type ReceivedFile struct {
	gorm.Model
	Processing     bool
	ChatID         int64 `gorm:"uniqueIndex:idx_chat_id_message_id;not null"`
	MessageID      int   `gorm:"uniqueIndex:idx_chat_id_message_id;not null"`
	ReplyMessageID int
	FileName       string
}

type User struct {
	gorm.Model
	UserID         int64 `gorm:"uniqueIndex"`
	Silent         bool
	DefaultStorage string
}
