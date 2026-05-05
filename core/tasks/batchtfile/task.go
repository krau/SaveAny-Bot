package batchtfile

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/pkg/metadata"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

var _ core.Executable = (*Task)(nil)

type TaskElement struct {
	ID        string
	Storage   storage.Storage
	Path      string
	File      tfile.TGFile
	localPath string
	stream    bool
	metadata  []byte
}

func (e *TaskElement) saveMetadata(ctx context.Context, actualPath string) error {
	if len(e.metadata) == 0 {
		return nil
	}
	_, err := e.Storage.Save(ctx, bytes.NewReader(e.metadata), actualPath+metadata.MetaSuffix)
	return err
}

type Task struct {
	ID           string
	ctx          context.Context
	elems        []TaskElement
	Progress     ProgressTracker
	IgnoreErrors bool // if true, errors during processing will be ignored
	downloaded   atomic.Int64
	totalSize    int64
	processing   map[string]TaskElementInfo
	processingMu sync.RWMutex
	failed       map[string]error // [TODO] errors for each element
}

// Title implements core.Exectable.
func (t *Task) Title() string {
	return fmt.Sprintf("[%s](%d files/%.2fMB)", t.Type(), len(t.elems), float64(t.totalSize)/(1024*1024))
}

func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeTgfiles
}

func NewTaskElement(
	ctx context.Context,
	stor storage.Storage,
	path string,
	file tfile.TGFile,
) (*TaskElement, error) {
	id := xid.New().String()

	var meta []byte
	if config.C().SaveMetadata {
		if fmsg, ok := file.(tfile.TGFileMessage); ok {
			m := metadata.BuildFromMessage(ctx, fmsg.Message(), file.Name(), file.Size())
			var err error
			meta, err = m.ToJSON()
			if err != nil {
				log.FromContext(ctx).Warnf("failed to marshal metadata: %s", err)
			}
		}
	}

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
			metadata:  meta,
		}, nil
	}
	return &TaskElement{
		ID:       id,
		Storage:  stor,
		Path:     path,
		File:     file,
		stream:   true,
		metadata: meta,
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
		ctx:        ctx,
		elems:      files,
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
