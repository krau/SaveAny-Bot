package telegraph

type TaskInfo interface {
	TaskID() string
	Phpath() string
	TotalPics() int
	Downloaded() int64
	StorageName() string
	StoragePath() string
}

func (t *Task) TaskID() string {
	return t.ID
}

func (t *Task) Phpath() string {
	return t.PhPath
}

func (t *Task) TotalPics() int {
	return t.totalpics
}

func (t *Task) Downloaded() int64 {
	return t.downloaded.Load()
}

func (t *Task) StorageName() string {
	return t.Stor.Name()
}

func (t *Task) StoragePath() string {
	return t.StorPath
}
