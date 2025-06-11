package tftask

type TaskInfo interface {
	FileName() string
	FileSize() int64
	StoragePath() string
	StorageName() string
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
