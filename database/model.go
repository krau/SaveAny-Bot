package database

import (
	"time"

	"gorm.io/gorm"

	"github.com/krau/SaveAny-Bot/pkg/enums/storagefilter"
)

type User struct {
	gorm.Model
	ChatID               int64 `gorm:"uniqueIndex;not null"`
	Silent               bool
	DefaultStorageID     uint // StorageRecord.ID
	DefaultDirID         uint // Dir.ID
	Dirs                 []Dir
	ApplyRule            bool
	Rules                []Rule
	WatchChats           []WatchChat
	WatchNotify          bool
	FilenameStrategy     string
	FilenameTemplate     string
	ConflictStrategy     string
	Enabled              bool               `gorm:"not null;default:true"`
	StorageFilterIDsJSON string             `gorm:"column:storage_filter_ids;type:text"` // JSON array of storage IDs
	StorageFilterMode    storagefilter.Mode `gorm:"size:16;not null;default:blacklist"`  // blacklist or whitelist
}

type WatchChat struct {
	gorm.Model
	UserID        uint // User's database ID (not chat ID)
	ChatID        int64
	MessageFilter string // filter type and expression, for example msgre:foo
}

type Dir struct {
	gorm.Model
	UserID    uint
	StorageID uint // StorageRecord.ID
	Path      string
}

type Rule struct {
	gorm.Model
	UserID    uint
	Type      string
	Content   string // rule content interpreted according to Type
	StorageID uint   // StorageRecord.ID
	DirPath   string
}

type Setting struct {
	Key       string `gorm:"primaryKey;size:64"`
	Value     string `gorm:"type:text;not null"`
	UpdatedAt time.Time
}

type StorageRecord struct {
	gorm.Model
	Name       string `gorm:"uniqueIndex;size:64;not null"`
	Type       string `gorm:"size:32;not null"`
	Enable     bool   `gorm:"not null"`
	ConfigJSON string `gorm:"column:config;type:text;not null"` // JSON storage configuration
}

func (StorageRecord) TableName() string { return "storages" }

// BotWorker is an optional helper bot used for Telegram storage uploads.
type BotWorker struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex;size:64;not null"`
	Token       string `gorm:"type:text;not null"`
	SessionPath string `gorm:"size:255"`
	Enabled     bool   `gorm:"not null"`
}

type TaskRecord struct {
	ID          uint   `gorm:"primaryKey"`
	TaskID      string `gorm:"uniqueIndex;size:32;not null"`
	Type        string `gorm:"size:32;not null"`
	Title       string
	Status      string `gorm:"size:16;index;not null"`
	StorageName string
	Path        string
	Source      string `gorm:"size:16;not null"`
	Error       string `gorm:"type:text"`
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	UpdatedAt   time.Time
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
