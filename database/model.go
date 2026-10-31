package database

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ChatID           int64 `gorm:"uniqueIndex;not null"`
	Silent           bool
	DefaultStorage   string
	DefaultDir       uint // Dir.ID
	Dirs             []Dir
	ApplyRule        bool
	Rules            []Rule
	WatchChats       []WatchChat
	WatchNotify      bool
	FilenameStrategy string
	FilenameTemplate string
	ConflictStrategy string
	Enabled          bool   `gorm:"not null;default:true"`
	StoragesJSON     string `gorm:"column:storages;type:text"`
	Blacklist        bool
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

type Setting struct {
	Key       string `gorm:"primaryKey;size:64"`
	Value     string `gorm:"type:text;not null"`
	UpdatedAt time.Time
}

type StorageRecord struct {
	gorm.Model
	Name   string `gorm:"uniqueIndex;size:64;not null"`
	Type   string `gorm:"size:32;not null"`
	Enable bool   `gorm:"not null"`
	Config string `gorm:"type:text;not null"`
}

func (StorageRecord) TableName() string { return "storages" }

type TaskRecord struct {
	ID         uint   `gorm:"primaryKey"`
	TaskID     string `gorm:"uniqueIndex;size:32;not null"`
	Type       string `gorm:"size:32;not null"`
	Title      string
	Status     string `gorm:"size:16;index;not null"`
	Storage    string
	Path       string
	Source     string `gorm:"size:16;not null"`
	Error      string `gorm:"type:text"`
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
	UpdatedAt  time.Time
}

type WebSession struct {
	ID        string `gorm:"primaryKey;size:64"`
	CSRF      string `gorm:"size:64;not null"`
	CreatedAt time.Time
	ExpiresAt time.Time `gorm:"index"`
}

type TaskFilter struct {
	Status string
	Type   string
	Limit  int
	Offset int
}
