package batchtfile

type TaskElementInfo interface {
	FileName() string
	FileSize() int64
	StoragePath() string
	StorageName() string
}

func (e *TaskElement) FileName() string {
	return e.File.Name()
}

func (e *TaskElement) FileSize() int64 {
	return e.File.Size()
}

func (e *TaskElement) StoragePath() string {
	return e.Path
}

func (e *TaskElement) StorageName() string {
	return e.Storage.Name()
}

type TaskInfo interface {
	TaskID() string
	TotalSize() int64
	Downloaded() int64
	Count() int
	Processing() []TaskElementInfo
}

func (t *Task) TaskID() string {
	return t.ID
}

func (t *Task) TotalSize() int64 {
	return t.totalSize
}

func (t *Task) Downloaded() int64 {
	return t.downloaded.Load()
}

func (t *Task) Count() int {
	return len(t.Elems)
}

func (t *Task) Processing() []TaskElementInfo {
	processing := make([]TaskElementInfo, 0, len(t.Elems))
	for _, elem := range t.processing {
		processing = append(processing, elem)
	}
	return processing
}
