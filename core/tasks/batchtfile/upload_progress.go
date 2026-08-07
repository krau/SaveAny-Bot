package batchtfile

import "context"

// UploadProgressTracker optionally extends a batch progress tracker with a
// distinct aggregate upload phase.
type UploadProgressTracker interface {
	OnUploadStart(ctx context.Context, info TaskInfo, total int64)
	OnUploadProgress(ctx context.Context, info TaskInfo, uploaded, total int64)
}

func (t *Task) startUpload(ctx context.Context) {
	tracker, ok := t.Progress.(UploadProgressTracker)
	if !ok {
		return
	}
	t.uploadMu.Lock()
	defer t.uploadMu.Unlock()
	t.uploadOnce.Do(func() {
		tracker.OnUploadStart(ctx, t, t.uploadTotalSize.Load())
	})
}

func (t *Task) uploadCallback(ctx context.Context, id string) func(uploaded, total int64) {
	return func(uploaded, total int64) {
		tracker, ok := t.Progress.(UploadProgressTracker)
		if !ok || uploaded < 0 {
			return
		}
		t.startUpload(ctx)
		if total > 0 && uploaded > total {
			uploaded = total
		}

		t.uploadMu.Lock()
		defer t.uploadMu.Unlock()
		if t.uploaded == nil {
			t.uploaded = make(map[string]int64)
		}
		previous, tracked := t.uploaded[id]
		if tracked && uploaded < previous {
			return
		}
		if !tracked || uploaded > previous {
			t.uploaded[id] = uploaded
		}
		var aggregate int64
		for _, current := range t.uploaded {
			aggregate += current
		}
		uploadTotal := t.uploadTotalSize.Load()
		if aggregate > uploadTotal {
			aggregate = uploadTotal
		}
		tracker.OnUploadProgress(ctx, t, aggregate, uploadTotal)
	}
}

func (t *Task) recordDownloadComplete(uploadSize int64) {
	t.uploadMu.Lock()
	defer t.uploadMu.Unlock()
	if uploadSize > 0 {
		t.uploadTotalSize.Add(uploadSize)
	}
	t.downloadedFiles.Add(1)
}
