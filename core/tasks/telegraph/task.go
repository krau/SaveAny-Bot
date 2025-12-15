package telegraph

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
	"github.com/krau/SaveAny-Bot/storage"
)

var _ core.Executable = (*Task)(nil)

type Task struct {
	ID       string
	Ctx      context.Context
	PhPath   string
	Pics     []string
	Stor     storage.Storage
	StorPath string
	client   *telegraph.Client
	progress ProgressTracker

	cannotStream bool
	totalpics    int
	downloaded   atomic.Int64
}

// Title implements core.Exectable.
func (t *Task) Title() string {
	return fmt.Sprintf("[%s](%s->%s:%s)", t.Type(), t.PhPath, t.Stor.Name(), t.StorPath)
}

func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeTphpics
}

func NewTask(
	id string,
	ctx context.Context,
	phPath string,
	pics []string,
	stor storage.Storage,
	storPath string,
	client *telegraph.Client,
	progress ProgressTracker,
) *Task {
	_, cannotStream := stor.(storage.StorageCannotStream)
	telegraph := &Task{
		ID:           id,
		Ctx:          ctx,
		PhPath:       phPath,
		Pics:         pics,
		Stor:         stor,
		StorPath:     storPath,
		client:       client,
		progress:     progress,
		cannotStream: cannotStream,
		totalpics:    len(pics),
		downloaded:   atomic.Int64{},
	}
	return telegraph
}
