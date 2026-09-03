package batchtfile

import (
	"context"
	"io"
	"time"

	"github.com/krau/SaveAny-Bot/common/tdler"
	"github.com/krau/SaveAny-Bot/common/utils/ioutil"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
)

// downloadToCache fetches elem.File into the element cache path, resuming
// from a partial .part download and reusing a complete cache file.
func (t *Task) downloadToCache(ctx context.Context, elem *TaskElement) error {
	return tdler.DownloadToCache(ctx, elem.File, elem.localPath, func(w io.WriterAt) io.WriterAt {
		return ioutil.NewProgressWriterAt(w, t.downloadCallback(ctx, elem))
	})
}

func (t *Task) downloadCallback(ctx context.Context, elem *TaskElement) func(int) {
	return func(n int) {
		t.recordItemDownload(elem.ID, int64(n), time.Now())
		downloaded := t.downloaded.Add(int64(n))
		t.notifyProgress(ctx)
		taskevent.Emit(ctx, taskevent.Event{
			TaskID:          t.ID,
			Phase:           taskevent.PhaseProgress,
			TotalBytes:      t.totalSize,
			DownloadedBytes: downloaded,
		})
	}
}
