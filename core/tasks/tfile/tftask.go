package tfile

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

var _ core.Executable = (*Task)(nil)
var _ core.DestinationInfo = (*Task)(nil)

type Task struct {
	ID        string
	ElementID string
	Done      []string
	Ctx       context.Context
	File      tfile.TGFile
	Storage   storage.Storage
	Path      string
	Progress  ProgressTracker
	stream    bool
	localPath string
	execErr   error
}

// Title implements core.Exectable.
func (t *Task) Title() string {
	return fmt.Sprintf("[%s](%s->%s:%s)", t.Type(), t.File.Name(), t.Storage.Name(), t.Path)
}

func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeTgfiles
}

func NewTGFileTask(
	id string,
	ctx context.Context,
	file tfile.TGFile,
	stor storage.Storage,
	path string,
	progress ProgressTracker,
) (*Task, error) {
	_, ok := stor.(storage.StorageCannotStream)
	if !config.C().Stream || ok {
		cachePath, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, fmt.Sprintf("%s_%s", id, fsutil.SafeBaseName(file.Name()))))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for cache: %w", err)
		}
		tfile := &Task{
			ID:        id,
			ElementID: id,
			Ctx:       ctx,
			File:      file,
			Storage:   stor,
			Path:      path,
			Progress:  progress,
			localPath: cachePath,
		}
		return tfile, nil
	}
	tfileTask := &Task{
		ID:        id,
		ElementID: id,
		Ctx:       ctx,
		File:      file,
		Storage:   stor,
		Path:      path,
		Progress:  progress,
		stream:    true,
	}
	return tfileTask, nil
}
