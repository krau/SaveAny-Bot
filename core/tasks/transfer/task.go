package transfer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

var _ core.Executable = (*Task)(nil)

type TaskElement struct {
	ID            string
	SourceStorage storage.Storage
	SourcePath    string
	FileInfo      storagetypes.FileInfo
	TargetStorage storage.Storage
	TargetPath    string
}

type Task struct {
	ID           string
	ctx          context.Context
	elems        []TaskElement
	Progress     ProgressTracker
	IgnoreErrors bool
	uploaded     atomic.Int64
	totalSize    int64
	processing   map[string]TaskElementInfo
	processingMu sync.RWMutex
	failed       map[string]error
	overwrite    bool // recovered: overwrite storage targets instead of uniquifying
	// doneList preserves element IDs persisted as done by an earlier run
	// (skipped from elems on recovery), so a re-persist does not lose them.
	doneList []string
	doneMu   sync.Mutex
}

// completedElementIDs returns the element IDs whose transfer finished, for
// persisting progress so recovery can skip them.
func (t *Task) completedElementIDs() []string {
	t.doneMu.Lock()
	defer t.doneMu.Unlock()
	return append([]string(nil), t.doneList...)
}

// persistElementDone records an element's completed transfer in the
// persisted payload so a restart does not transfer it again. Serialized so
// concurrent element completions do not clobber each other's payload updates.
func (t *Task) persistElementDone(ctx context.Context, elemID string) {
	t.doneMu.Lock()
	defer t.doneMu.Unlock()
	for _, id := range t.doneList {
		if id == elemID {
			return
		}
	}
	if err := core.AppendTaskDone(ctx, t.ID, elemID); err != nil {
		log.FromContext(ctx).Warnf("Failed to persist element completion %s: %v", elemID, err)
		return
	}
	t.doneList = append(t.doneList, elemID)
}

// Title implements core.Executable.
func (t *Task) Title() string {
	return fmt.Sprintf("[%s](%d files/%.2fMB)", t.Type(), len(t.elems), float64(t.totalSize)/(1024*1024))
}

// Type implements core.Executable.
func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeTransfer
}

// TaskID implements core.Executable.
func (t *Task) TaskID() string {
	return t.ID
}

func NewTaskElement(
	sourceStorage storage.Storage,
	fileInfo storagetypes.FileInfo,
	targetStorage storage.Storage,
	targetPath string,
) *TaskElement {
	id := xid.New().String()
	return &TaskElement{
		ID:            id,
		SourceStorage: sourceStorage,
		SourcePath:    fileInfo.Path,
		FileInfo:      fileInfo,
		TargetStorage: targetStorage,
		TargetPath:    targetPath,
	}
}

func NewTransferTask(
	id string,
	ctx context.Context,
	elems []TaskElement,
	progress ProgressTracker,
	ignoreErrors bool,
) *Task {
	task := &Task{
		ID:       id,
		ctx:      ctx,
		elems:    elems,
		Progress: progress,
		uploaded: atomic.Int64{},
		totalSize: func() int64 {
			var total int64
			for _, elem := range elems {
				total += elem.FileInfo.Size
			}
			return total
		}(),
		processing:   make(map[string]TaskElementInfo),
		IgnoreErrors: ignoreErrors,
		failed:       make(map[string]error),
	}
	return task
}
