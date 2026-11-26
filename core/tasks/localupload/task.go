package localupload

import (
	"context"
	"fmt"
	"path"
	"sync"
	"sync/atomic"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/storage"
)

var _ core.Executable = (*Task)(nil)
var _ core.DestinationInfo = (*Task)(nil)
var _ core.ParamSource = (*Task)(nil)
var _ core.ResultReporter = (*Task)(nil)

// Task uploads a file staged by the web API to a configured storage.
type Task struct {
	ID          string
	Name        string
	Size        int64
	Storage     storage.Storage
	Path        string
	Overwrite   bool
	stagingRoot string

	uploaded   atomic.Int64
	mu         sync.Mutex
	execErr    string
	result     core.ItemStatus
	actualPath string
}

func NewTask(
	id string,
	stagingRoot string,
	name string,
	size int64,
	stor storage.Storage,
	targetPath string,
	overwrite bool,
) *Task {
	return &Task{
		ID:          id,
		stagingRoot: stagingRoot,
		Name:        name,
		Size:        size,
		Storage:     stor,
		Path:        targetPath,
		Overwrite:   overwrite,
		result:      core.ItemStatusCompleted,
	}
}

func (t *Task) Title() string {
	return fmt.Sprintf("[%s](%s->%s:%s)", t.Type(), t.Name, t.Storage.Name(), t.Path)
}

func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeUpload
}

func (t *Task) TaskID() string {
	return t.ID
}

func (t *Task) StorageName() string {
	return t.Storage.Name()
}

func (t *Task) StoragePath() string {
	return path.Join(t.Path, t.Name)
}

func (t *Task) TotalSize() int64 {
	return t.Size
}

func (t *Task) Uploaded() int64 {
	return t.uploaded.Load()
}

func (t *Task) setResult(ctx context.Context, actualPath string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.actualPath = actualPath
	switch {
	case err == nil:
		t.result = core.ItemStatusCompleted
		t.execErr = ""
	case ctx.Err() != nil:
		t.result = core.ItemStatusCancelled
		t.execErr = err.Error()
	default:
		t.result = core.ItemStatusFailed
		t.execErr = err.Error()
	}
}
