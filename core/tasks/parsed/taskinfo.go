package parsed

type TaskInfo interface {
	TaskID() string
	Site() string
	TotalResources() int64
	Downloaded() int64
	TotalBytes() int64
	DownloadedBytes() int64
	Processing() map[string]ResourceInfo
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

func (t *Task) TotalBytes() int64 {
	return t.totalBytes
}

func (t *Task) DownloadedBytes() int64 {
	return t.downloadedBytes.Load()
}

func (t *Task) Processing() map[string]ResourceInfo {
	t.processingMu.RLock()
	defer t.processingMu.RUnlock()
	return t.processing
}

type ResourceInfo interface {
	FileName() string
	FileSize() int64
}
