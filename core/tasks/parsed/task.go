package parsed

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/parser"
	"github.com/krau/SaveAny-Bot/storage"
)

type Task struct {
	ID         string
	Ctx        context.Context
	Stor       storage.Storage
	StorPath   string
	item       *parser.Item
	httpClient *http.Client
	progress   ProgressTracker

	totalResources int64
	downloaded     atomic.Int64 // downloaded resources count
}

func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeParseditem
}

func (t *Task) TaskID() string {
	return t.ID
}

func NewTask(
	id string,
	ctx context.Context,
	stor storage.Storage,
	storPath string,
	item *parser.Item,
	progressTracker ProgressTracker,
) *Task {
	client := &http.Client{
		Transport: &http.Transport{
			// [TODO] configure it via config
			Proxy: http.ProxyFromEnvironment,
		},
	}
	return &Task{
		ID:             id,
		Ctx:            ctx,
		Stor:           stor,
		StorPath:       storPath,
		item:           item,
		totalResources: int64(len(item.Resources)),
		downloaded:     atomic.Int64{},
		httpClient:     client,
		progress:       progressTracker,
	}
}
