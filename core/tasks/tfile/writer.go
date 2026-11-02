package tfile

import (
	"context"
	"io"
	"sync/atomic"

	"github.com/krau/SaveAny-Bot/common/utils/ioutil"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
)

type ProgressWriterAt struct {
	ctx        context.Context
	progress   ProgressTracker
	downloaded *atomic.Int64
	total      int64
	info       TaskInfo
	writer     *ioutil.ProgressWriterAt
}

func (w *ProgressWriterAt) WriteAt(p []byte, off int64) (int, error) {
	return w.writer.WriteAt(p, off)
}

func newWriterAt(
	ctx context.Context,
	wrAt io.WriterAt,
	progress ProgressTracker,
	taskInfo TaskInfo,
) *ProgressWriterAt {
	w := &ProgressWriterAt{
		ctx:        ctx,
		progress:   progress,
		downloaded: &atomic.Int64{},
		total:      taskInfo.FileSize(),
		info:       taskInfo,
	}
	w.writer = ioutil.NewProgressWriterAt(wrAt, w.record)
	return w
}

func (w *ProgressWriterAt) record(n int) {
	downloaded := w.downloaded.Add(int64(n))
	if w.progress != nil {
		w.progress.OnProgress(w.ctx, w.info, downloaded, w.total)
	}
	taskevent.Emit(w.ctx, taskevent.Event{
		TaskID:          w.info.TaskID(),
		Phase:           taskevent.PhaseProgress,
		TotalBytes:      w.total,
		DownloadedBytes: downloaded,
	})
}

type ProgressWriter struct {
	ctx        context.Context
	progress   ProgressTracker
	downloaded *atomic.Int64
	total      int64
	info       TaskInfo
	writer     *ioutil.ProgressWriter
}

func (w *ProgressWriter) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}

func newWriter(
	ctx context.Context,
	wr io.Writer,
	progress ProgressTracker,
	taskInfo TaskInfo,
) *ProgressWriter {
	w := &ProgressWriter{
		ctx:        ctx,
		progress:   progress,
		downloaded: &atomic.Int64{},
		total:      taskInfo.FileSize(),
		info:       taskInfo,
	}
	w.writer = ioutil.NewProgressWriter(wr, w.record)
	return w
}

func (w *ProgressWriter) record(n int) {
	downloaded := w.downloaded.Add(int64(n))
	if w.progress != nil {
		w.progress.OnProgress(w.ctx, w.info, downloaded, w.total)
	}
	taskevent.Emit(w.ctx, taskevent.Event{
		TaskID:          w.info.TaskID(),
		Phase:           taskevent.PhaseProgress,
		TotalBytes:      w.total,
		DownloadedBytes: downloaded,
	})
}
