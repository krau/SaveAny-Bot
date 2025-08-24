package parsed

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/krau/SaveAny-Bot/common/utils/netutil"
	"github.com/krau/SaveAny-Bot/config"
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
	stream     bool

	totalResources  int64
	downloaded      atomic.Int64 // downloaded resources count
	totalBytes      int64        // total bytes to download
	downloadedBytes atomic.Int64 // downloaded bytes count
	processing      map[string]ResourceInfo
	processingMu    sync.RWMutex
	failed          map[string]error // [TODO] errors for each resource
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
	client := netutil.DefaultParserHTTPClient()
	_, ok := stor.(storage.StorageCannotStream)
	stream := config.C().Stream && !ok
	return &Task{
		ID:             id,
		Ctx:            ctx,
		Stor:           stor,
		StorPath:       storPath,
		item:           item,
		totalResources: int64(len(item.Resources)),
		downloaded:     atomic.Int64{},
		totalBytes: func() int64 {
			var total int64
			for _, res := range item.Resources {
				if res.Size < 0 {
					continue // skip resources with unknown size
				}
				total += res.Size
			}
			return total
		}(),
		stream:          stream,
		downloadedBytes: atomic.Int64{},
		httpClient:      client,
		progress:        progressTracker,
		processing:      make(map[string]ResourceInfo),
		processingMu:    sync.RWMutex{},
		failed:          make(map[string]error),
	}
}
