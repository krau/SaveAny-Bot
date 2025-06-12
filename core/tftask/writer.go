package tftask

import (
	"context"
	"io"
	"sync/atomic"
)

type ProgressWriterAt struct {
	ctx        context.Context
	wrAt       io.WriterAt
	progress   ProgressTracker
	downloaded *atomic.Int64
	total      int64
	info       TaskInfo
}

func (w *ProgressWriterAt) WriteAt(p []byte, off int64) (int, error) {
	at, err := w.wrAt.WriteAt(p, off)
	if err != nil {
		return 0, err
	}
	w.progress.OnProgress(w.ctx, w.info, w.downloaded.Add(int64(at)), w.total)
	return at, nil
}

func newWriterAt(
	ctx context.Context,
	wrAt io.WriterAt,
	progress ProgressTracker,
	taskInfo TaskInfo,
) *ProgressWriterAt {
	return &ProgressWriterAt{
		ctx:        ctx,
		progress:   progress,
		downloaded: &atomic.Int64{},
		total:      taskInfo.FileSize(),
		wrAt:       wrAt,
		info:       taskInfo,
	}
}
