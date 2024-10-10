package model

import (
	"gorm.io/gorm"
)

type ReceivedFile struct {
	gorm.Model
	// FileUniqueID 和 FileID 在数据库中均不具有唯一性
	// 如需确定唯一一行, 使用 ChatID + MessageID
	FileUniqueID   string `gorm:"index"`
	FileID         string `gorm:"index"`
	Processing     bool
	FileName       string
	FilePath       string
	FileSize       int64
	MediaGroupID   string
	ChatID         int64
	MessageID      int
	ReplyMessageID int
}

type User struct {
	gorm.Model
	UserID         int64 `gorm:"uniqueIndex"`
	Silent         bool
	DefaultStorage string
}
