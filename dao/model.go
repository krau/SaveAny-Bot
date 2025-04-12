package dao

import (
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
	IsTelegraph    bool
	TelegraphURL   string
}

type User struct {
	gorm.Model
	ChatID         int64 `gorm:"uniqueIndex;not null"`
	Silent         bool
	DefaultStorage string // Default storage name
	Dirs           []Dir
	ApplyRule      bool
	Rules          []Rule
}

type Dir struct {
	gorm.Model
	UserID      uint
	StorageName string
	Path        string
}

type CallbackData struct {
	gorm.Model
	Data string
}

type Rule struct {
	gorm.Model
	UserID      uint
	Type        string
	Data        string
	StorageName string
	DirPath     string
}
