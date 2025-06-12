package database

import (
	"gorm.io/gorm"
)

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

type Rule struct {
	gorm.Model
	UserID      uint
	Type        string
	Data        string
	StorageName string
	DirPath     string
}
