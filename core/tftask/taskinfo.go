package tftask

type TaskInfo interface {
	FileName() string
	FileSize() int64
	StoragePath() string
	StorageName() string
	TaskID() string
}

// type taskInfoImpl struct {
// 	ID          string
// 	Filename    string
// 	Filesize    int64
// 	Storagename string
// 	Storagepath string
// }

// func (t *taskInfoImpl) TaskID() string {
// 	return t.ID
// }
// func (t *taskInfoImpl) FileName() string {
// 	return t.Filename
// }
// func (t *taskInfoImpl) FileSize() int64 {
// 	return t.Filesize
// }
// func (t *taskInfoImpl) StoragePath() string {
// 	return t.Storagepath
// }
// func (t *taskInfoImpl) StorageName() string {
// 	return t.Storagename
// }

func (t *TGFileTask) TaskID() string {
	return t.ID
}

func (t *TGFileTask) FileName() string {
	return t.File.Name()
}

func (t *TGFileTask) FileSize() int64 {
	return t.File.Size()
}

func (t *TGFileTask) StoragePath() string {
	return t.Path
}

func (t *TGFileTask) StorageName() string {
	return t.Storage.Name()
}
