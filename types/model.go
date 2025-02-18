package types

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ReceivedFile struct {
	gorm.Model
	Processing bool
	// Which chat the file is from
	ChatID int64 `gorm:"uniqueIndex:idx_chat_id_message_id;not null"`
	// Which message the file is from
	MessageID      int `gorm:"uniqueIndex:idx_chat_id_message_id;not null"`
	ReplyMessageID int
	ReplyChatID    int64
	FileName       string
}

type User struct {
	gorm.Model
	ChatID           int64 `gorm:"uniqueIndex"` // Telegram user ID
	Silent           bool
	DefaultStorageID uint
	Storages         []*StorageModel `gorm:"many2many:user_storages;"`
}

type StorageModel struct {
	gorm.Model
	Type   string
	Name   string // just for display
	Desc   string
	Active bool
	Config datatypes.JSON
	Users  []*User `gorm:"many2many:user_storages;"`
}
