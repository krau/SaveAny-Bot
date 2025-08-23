package batchtfile

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

type TaskElement struct {
	ID        string
	Storage   storage.Storage
	Path      string
	File      tfile.TGFile
	localPath string
	stream    bool
}

type Task struct {
	ID           string
	Ctx          context.Context
	Elems        []TaskElement
	Progress     ProgressTracker
	IgnoreErrors bool // if true, errors during processing will be ignored
	downloaded   atomic.Int64
	totalSize    int64
	processing   map[string]TaskElementInfo
	processingMu sync.RWMutex
	failed       map[string]error // [TODO] errors for each element
}

func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeTgfiles
}

func NewTaskElement(
	stor storage.Storage,
	path string,
	file tfile.TGFile,
) (*TaskElement, error) {
	id := xid.New().String()
	_, ok := stor.(storage.StorageCannotStream)
	if !config.C().Stream || ok {
		cachePath, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, fmt.Sprintf("%s_%s", id, file.Name())))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for cache: %w", err)
		}
		return &TaskElement{
			ID:        id,
			Storage:   stor,
			Path:      path,
			File:      file,
			localPath: cachePath,
		}, nil
	}
	return &TaskElement{
		ID:      id,
		Storage: stor,
		Path:    path,
		File:    file,
		stream:  true,
	}, nil
}

func NewBatchTGFileTask(
	id string,
	ctx context.Context,
	files []TaskElement,
	progress ProgressTracker,
	ignoreErrors bool,
) *Task {
	task := &Task{
		ID:         id,
		Ctx:        ctx,
		Elems:      files,
		Progress:   progress,
		downloaded: atomic.Int64{},
		totalSize: func() int64 {
			var total int64
			for _, elem := range files {
				total += elem.File.Size()
			}
			return total
		}(),
		processing:   make(map[string]TaskElementInfo),
		IgnoreErrors: ignoreErrors,
		processingMu: sync.RWMutex{},
		failed:       make(map[string]error),
	}
	return task
}
