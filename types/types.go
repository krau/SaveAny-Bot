package types

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gotd/td/tg"
)

type TaskStatus string

var (
	Pending   TaskStatus = "pending"
	Succeeded TaskStatus = "succeeded"
	Failed    TaskStatus = "failed"
	Canceled  TaskStatus = "canceled"
)

type StorageType string

var (
	StorageAll StorageType = "all"
	Local      StorageType = "local"
	Webdav     StorageType = "webdav"
	Alist      StorageType = "alist"
)

var StorageTypes = []StorageType{Local, Alist, Webdav, StorageAll}

type Task struct {
	Ctx         context.Context
	Error       error
	Status      TaskStatus
	File        *File
	Storage     StorageType
	StoragePath string
	StartTime   time.Time

	FileMessageID  int
	FileChatID     int64
	ReplyMessageID int
	ReplyChatID    int64
}

func (t Task) String() string {
	return fmt.Sprintf("[%d:%d]:%s", t.FileChatID, t.FileMessageID, t.File.FileName)
}

func (t Task) FileName() string {
	return t.File.FileName
}

type File struct {
	Location tg.InputFileLocationClass
	FileSize int64
	FileName string
}

func (f File) Hash() string {
	locationBytes := []byte(f.Location.String())
	fileSizeBytes := []byte(fmt.Sprintf("%d", f.FileSize))
	fileNameBytes := []byte(f.FileName)

	structBytes := append(locationBytes, fileSizeBytes...)
	structBytes = append(structBytes, fileNameBytes...)

	hash := md5.New()
	hash.Write(structBytes)
	hashBytes := hash.Sum(nil)

	return hex.EncodeToString(hashBytes)
}
