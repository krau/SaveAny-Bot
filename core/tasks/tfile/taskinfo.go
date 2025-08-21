package tfile

type TaskInfo interface {
	TaskID() string
	FileName() string
	FileSize() int64
	StoragePath() string
	StorageName() string
}

func (t *Task) TaskID() string {
	return t.ID
}

func (t *Task) FileName() string {
	return t.File.Name()
}

func (t *Task) FileSize() int64 {
	return t.File.Size()
}

func (t *Task) StoragePath() string {
	return t.Path
}

func (t *Task) StorageName() string {
	return t.Storage.Name()
}
