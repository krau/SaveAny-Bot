package types

import (
	"crypto/md5"
	"encoding/hex"

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
	ChatID           int64 `gorm:"uniqueIndex;not null"`
	Silent           bool
	DefaultStorageID uint
	Storages         []*StorageModel `gorm:"many2many:user_storages;"`
}

type StorageModel struct {
	gorm.Model
	Type   string
	Config datatypes.JSON
	Active bool
	Users  []*User `gorm:"many2many:user_storages;"`
	Hash   string  `gorm:"uniqueIndex"`
	// just for display
	Name string `gorm:"not null"`
	Desc string
}

func (s *StorageModel) GenHash() string {
	if s.Type == "" || s.Config == nil {
		return ""
	}
	typeBytes := []byte(s.Type)
	configBytes := s.Config
	structBytes := append(typeBytes, configBytes...)
	hash := md5.New()
	hash.Write(structBytes)
	hashBytes := hash.Sum(nil)
	return hex.EncodeToString(hashBytes)
}
