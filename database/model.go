package database

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ChatID           int64 `gorm:"uniqueIndex;not null"`
	Silent           bool
	DefaultStorage   string
	Dirs             []Dir
	ApplyRule        bool
	Rules            []Rule
	WatchChats       []WatchChat
	FilenameStrategy string
}

type WatchChat struct {
	gorm.Model
	UserID uint // User's database ID (not chat ID)
	ChatID int64
	Filter string
}

type Dir struct {
	gorm.Model
	UserID      uint
	StorageName string
	Path        string
}

type Rule struct {
	gorm.Model
	UserID      uint
	Type        string
	Data        string
	StorageName string
	DirPath     string
}
