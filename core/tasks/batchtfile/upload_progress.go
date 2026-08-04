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
	t.uploadOnce.Do(func() {
		tracker.OnUploadStart(ctx, t, t.totalSize)
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
		if t.uploaded == nil {
			t.uploaded = make(map[string]int64)
		}
		t.uploaded[id] = uploaded
		var aggregate int64
		for _, current := range t.uploaded {
			aggregate += current
		}
		t.uploadMu.Unlock()

		if aggregate > t.totalSize {
			aggregate = t.totalSize
		}
		tracker.OnUploadProgress(ctx, t, aggregate, t.totalSize)
	}
}
