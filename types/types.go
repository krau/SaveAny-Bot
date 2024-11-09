package types

import (
	"context"
	"fmt"

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

	MessageID      int
	ChatID         int64
	ReplyMessageID int
}

func (t Task) String() string {
	return fmt.Sprintf("[%d:%d]:%s", t.ChatID, t.MessageID, t.File.FileName)
}

func (t Task) FileName() string {
	return t.File.FileName
}

type File struct {
	Location *tg.InputDocumentFileLocation
	FileSize int64
	FileName string
	MimeType string
	ID       int64
}
