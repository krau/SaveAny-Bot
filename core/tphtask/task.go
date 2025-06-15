package tphtask

import (
	"context"
	"sync/atomic"

	"github.com/krau/SaveAny-Bot/pkg/telegraph"
	"github.com/krau/SaveAny-Bot/storage"
)

type Task struct {
	ID       string
	Ctx      context.Context
	PhPath   string
	Pics     []string
	Stor     storage.Storage
	StorPath string
	client   *telegraph.Client
	progress ProgressTracker

	totalpics  int
	downloaded atomic.Int64
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
	tphtask := &Task{
		ID:         id,
		Ctx:        ctx,
		PhPath:     phPath,
		Pics:       pics,
		Stor:       stor,
		StorPath:   storPath,
		client:     client,
		progress:   progress,
		totalpics:  len(pics),
		downloaded: atomic.Int64{},
	}
	return tphtask
}
