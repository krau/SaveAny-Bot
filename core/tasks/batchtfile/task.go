package batchtfile

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

var _ core.Executable = (*Task)(nil)

type TaskElement struct {
	ID              string
	Storage         storage.Storage
	Path            string
	File            tfile.TGFile
	localPath       string
	stream          bool
	sourceGroupKey  string
	sourceCaption   string
	preserveCaption bool
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
	uploadOnce   sync.Once
	uploadMu     sync.Mutex
	uploaded     map[string]int64
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
	stor storage.Storage,
	path string,
	file tfile.TGFile,
) (*TaskElement, error) {
	id := xid.New().String()
	groupKey, caption, preserveCaption := sourceMetadata(file)
	_, ok := stor.(storage.StorageCannotStream)
	if !config.C().Stream || ok {
		cachePath, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, fmt.Sprintf("%s_%s", id, file.Name())))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for cache: %w", err)
		}
		return &TaskElement{
			ID:              id,
			Storage:         stor,
			Path:            path,
			File:            file,
			localPath:       cachePath,
			sourceGroupKey:  groupKey,
			sourceCaption:   caption,
			preserveCaption: preserveCaption,
		}, nil
	}
	return &TaskElement{
		ID:              id,
		Storage:         stor,
		Path:            path,
		File:            file,
		stream:          true,
		sourceGroupKey:  groupKey,
		sourceCaption:   caption,
		preserveCaption: preserveCaption,
	}, nil
}

func sourceMetadata(file tfile.TGFile) (groupKey, caption string, preserveCaption bool) {
	messageFile, ok := file.(tfile.TGFileMessage)
	if !ok || messageFile.Message() == nil {
		return "", "", false
	}
	msg := messageFile.Message()
	groupID, grouped := msg.GetGroupedID()
	if !grouped || groupID == 0 {
		return "", "", false
	}
	chatID := tgutil.ChatIdFromPeer(msg.GetPeerID())
	return fmt.Sprintf("%T:%d:%d", msg.GetPeerID(), chatID, groupID), msg.GetMessage(), true
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
		uploaded:     make(map[string]int64),
		IgnoreErrors: ignoreErrors,
		processingMu: sync.RWMutex{},
		failed:       make(map[string]error),
	}
	return task
}
