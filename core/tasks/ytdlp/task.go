package ytdlp

import (
	"context"
	"fmt"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/storage"
)

var _ core.Executable = (*Task)(nil)

type Task struct {
	ID       string
	ctx      context.Context
	URLs     []string
	Flags    []string
	Storage  storage.Storage
	StorPath string
	Progress ProgressTracker
}

// Title implements core.Executable.
func (t *Task) Title() string {
	urlCount := len(t.URLs)
	if urlCount == 1 {
		return fmt.Sprintf("[%s](%s->%s:%s)", t.Type(), t.URLs[0], t.Storage.Name(), t.StorPath)
	}
	return fmt.Sprintf("[%s](%d URLs->%s:%s)", t.Type(), urlCount, t.Storage.Name(), t.StorPath)
}

// Type implements core.Executable.
func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeYtdlp
}

// TaskID implements core.Executable.
func (t *Task) TaskID() string {
	return t.ID
}

func NewTask(
	id string,
	ctx context.Context,
	urls []string,
	flags []string,
	stor storage.Storage,
	storPath string,
	progressTracker ProgressTracker,
) *Task {
	return &Task{
		ID:       id,
		ctx:      ctx,
		URLs:     urls,
		Flags:    flags,
		Storage:  stor,
		StorPath: storPath,
		Progress: progressTracker,
	}
}
