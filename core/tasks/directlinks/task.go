package directlinks

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/storage"
)

type File struct {
	Name string
	URL  string
	Size int64
}

func (f *File) FileName() string {
	return f.Name
}

func (f *File) FileSize() int64 {
	return f.Size
}

var _ core.Executable = (*Task)(nil)

type Task struct {
	ID       string
	ctx      context.Context
	files    []*File
	Storage  storage.Storage
	StorPath string
	Progress ProgressTracker

	client          *http.Client // [TODO] parallel download
	stream          bool
	totalBytes      int64            // total bytes to download
	downloadedBytes atomic.Int64     // downloaded bytes
	totalFiles      int64            // total files to download
	downloaded      atomic.Int64     // downloaded files count
	processing      map[string]*File // {"url": File}
	processingMu    sync.RWMutex
	failed          map[string]error // [TODO] errors for each file
}

// Title implements core.Exectable.
func (t *Task) Title() string {
	return fmt.Sprintf("[%s](%s...->%s:%s)", t.Type(), t.files[0].Name, t.Storage.Name(), t.StorPath)
}

// DownloadedBytes implements TaskInfo.
func (t *Task) DownloadedBytes() int64 {
	return t.downloadedBytes.Load()
}

// Processing implements TaskInfo.
func (t *Task) Processing() []FileInfo {
	t.processingMu.RLock()
	defer t.processingMu.RUnlock()
	infos := make([]FileInfo, 0, len(t.processing))
	for _, f := range t.processing {
		infos = append(infos, f)
	}
	return infos
}

// StorageName implements TaskInfo.
func (t *Task) StorageName() string {
	return t.Storage.Name()
}

// StoragePath implements TaskInfo.
func (t *Task) StoragePath() string {
	if len(t.files) == 1 {
		return t.StorPath + "/" + t.files[0].Name
	}
	return t.StorPath
}

// TotalBytes implements TaskInfo.
func (t *Task) TotalBytes() int64 {
	return t.totalBytes
}

// TotalFiles implements TaskInfo.
func (t *Task) TotalFiles() int {
	return int(t.totalFiles)
}

func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeDirectlinks
}

func (t *Task) TaskID() string {
	return t.ID
}

func NewTask(
	id string,
	ctx context.Context,
	links []string,
	stor storage.Storage,
	storPath string,
	progressTracker ProgressTracker,
) *Task {
	_, ok := stor.(storage.StorageCannotStream)
	stream := config.C().Stream && !ok
	files := make([]*File, 0, len(links))
	for _, link := range links {
		files = append(files, &File{
			URL: link,
		})
	}
	return &Task{
		ID:           id,
		ctx:          ctx,
		files:        files,
		Storage:      stor,
		StorPath:     storPath,
		Progress:     progressTracker,
		stream:       stream,
		client:       http.DefaultClient,
		processing:   make(map[string]*File),
		processingMu: sync.RWMutex{},
		failed:       make(map[string]error),
		totalFiles:   int64(len(files)),
	}
}
