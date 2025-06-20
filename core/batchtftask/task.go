package batchtftask

import (
	"context"
	"fmt"
	"path/filepath"
	"sync/atomic"

	"github.com/krau/SaveAny-Bot/common/tdler"
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
	client       tdler.Client
	totalSize    int64
	processing   map[string]TaskElementInfo
	failed       map[string]error // errors for each element
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
	if !config.Cfg.Stream || ok {
		cachePath, err := filepath.Abs(filepath.Join(config.Cfg.Temp.BasePath, fmt.Sprintf("%s_%s", id, file.Name())))
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
	client tdler.Client,
	progress ProgressTracker,
	ignoreErrors bool,
) *Task {
	task := &Task{
		ID:         id,
		Ctx:        ctx,
		client:     client,
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
		failed:       make(map[string]error),
	}
	return task
}
