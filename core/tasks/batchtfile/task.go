package batchtfile

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/metadata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

var _ core.Executable = (*Task)(nil)
var _ core.DestinationInfo = (*Task)(nil)

type TaskElement struct {
	ID               string
	Storage          storage.Storage
	Path             string
	File             tfile.TGFile
	localPath        string
	stream           bool
	sourceGroupKey   string
	sourceCaption    string
	preserveCaption  bool
	metadataJSON     string
	metadataVersion  int
	metadataGroupKey string
}

type Task struct {
	ID              string
	Done            []string
	ctx             context.Context
	elems           []TaskElement
	Progress        ProgressTracker
	IgnoreErrors    bool
	downloaded      atomic.Int64
	totalSize       int64
	uploadTotalSize atomic.Int64
	processing      map[string]TaskElementInfo
	processingMu    sync.RWMutex
	itemStates      []itemProgressState
	itemIndex       map[string]int
	itemMu          sync.RWMutex
	uploadOnce      sync.Once
	uploadMu        sync.Mutex
	uploaded        map[string]int64
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
	return NewTaskElementWithID(xid.New().String(), stor, path, file)
}

func NewTaskElementWithID(
	id string,
	stor storage.Storage,
	path string,
	file tfile.TGFile,
) (*TaskElement, error) {
	groupKey, caption, preserveCaption := sourceMetadata(file)
	_, ok := stor.(storage.StorageCannotStream)
	if !config.C().Stream || ok {
		cachePath, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, fmt.Sprintf("%s_%s", id, fsutil.SafeBaseName(file.Name()))))
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
	itemStates, itemIndex := newItemProgressStates(files)
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
		itemStates:   itemStates,
		itemIndex:    itemIndex,
		uploaded:     make(map[string]int64),
		IgnoreErrors: ignoreErrors,
		processingMu: sync.RWMutex{},
	}
	if config.C().Metadata.Enabled {
		for i := range task.elems {
			task.elems[i].metadataJSON, task.elems[i].metadataVersion = buildElementMetadata(ctx, task.elems[i].File)
			task.elems[i].metadataGroupKey = metadataGroupKey(task.elems[i].metadataJSON)
		}
	}
	return task
}

func buildElementMetadata(ctx context.Context, file tfile.TGFile) (string, int) {
	messageFile, ok := file.(tfile.TGFileMessage)
	if !ok || messageFile.Message() == nil {
		return "", 0
	}
	metaJSON, version, err := metadata.BuildJSON(ctx, messageFile.Message(), file.Name(), file.Size(), config.C().Metadata.IncludeText)
	if err != nil {
		log.FromContext(ctx).Warnf("Failed to build task metadata: %v", err)
		return "", 0
	}
	return metaJSON, version
}

func (e *TaskElement) saveMetadataSidecar(ctx context.Context, actualPath string) {
	if !config.C().Metadata.Sidecar || e.metadataJSON == "" || actualPath == "" {
		return
	}
	if err := storage.WriteMetadataSidecar(ctx, e.Storage, actualPath, []byte(e.metadataJSON)); err != nil {
		log.FromContext(ctx).Warnf("Failed to save metadata sidecar for %s: %v", actualPath, err)
	}
}

func metadataGroupKey(raw string) string {
	if raw == "" {
		return ""
	}
	var meta struct {
		Message struct {
			GroupedID int64 `json:"grouped_id"`
			Chat      struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(raw), &meta); err != nil || meta.Message.GroupedID == 0 {
		return ""
	}
	return fmt.Sprintf("metadata:%d:%d", meta.Message.Chat.ID, meta.Message.GroupedID)
}

func (e *TaskElement) executionGroupKey() string {
	if e.sourceGroupKey != "" {
		return e.sourceGroupKey
	}
	return e.metadataGroupKey
}
