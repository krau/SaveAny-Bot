package types

import "context"

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
	FileName    string
	Storage     StorageType
	StoragePath string

	MessageID      int32
	ChatID         int64
	ReplyMessageID int32
}

func (t *Task) String() string {
	return t.FileName
}
