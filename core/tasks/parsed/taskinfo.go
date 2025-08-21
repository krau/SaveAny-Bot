package parsed

type TaskInfo interface {
	TaskID() string
	Site() string
	TotalResources() int64
	Downloaded() int64
	StorageName() string
	StoragePath() string
}

func (t *Task) StoragePath() string {
	return t.StorPath
}
func (t *Task) TotalResources() int64 {
	return t.totalResources
}

func (t *Task) Downloaded() int64 {
	return t.downloaded.Load()
}

func (t *Task) StorageName() string {
	return t.Stor.Name()
}

func (t *Task) Site() string {
	return t.item.Site
}
