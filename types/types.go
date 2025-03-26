package types

type TaskStatus string

const (
	Pending   TaskStatus = "pending"
	Succeeded TaskStatus = "succeeded"
	Failed    TaskStatus = "failed"
	Canceled  TaskStatus = "canceled"
)

type StorageType string

const (
	StorageTypeLocal  StorageType = "local"
	StorageTypeWebdav StorageType = "webdav"
	StorageTypeAlist  StorageType = "alist"
	StorageTypeMinio  StorageType = "minio"
)

var StorageTypes = []StorageType{StorageTypeLocal, StorageTypeAlist, StorageTypeWebdav, StorageTypeMinio}
var StorageTypeDisplay = map[StorageType]string{
	StorageTypeLocal:  "本地磁盘",
	StorageTypeWebdav: "WebDAV",
	StorageTypeAlist:  "Alist",
	StorageTypeMinio:  "Minio",
}

type ContextKey string

const (
	ContextKeyContentLength ContextKey = "content-length"
)
