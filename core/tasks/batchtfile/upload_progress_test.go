package batchtfile

import (
	"context"
	"sync"
	"testing"
	"time"
)

type uploadProgressRecorder struct {
	mu         sync.Mutex
	starts     int
	startTotal int64
	uploaded   int64
	total      int64
}

func TestShouldUpdateUploadProgress(t *testing.T) {
	tests := []struct {
		name        string
		total       int64
		uploaded    int64
		lastPercent int
		elapsed     time.Duration
		want        bool
	}{
		{name: "invalid total", total: 0, uploaded: 1, want: false},
		{name: "no uploaded bytes", total: 100, uploaded: 0, want: false},
		{name: "percentage threshold", total: 100 << 20, uploaded: 10 << 20, elapsed: uploadProgressMinInterval, want: true},
		{name: "percentage threshold rate limited", total: 100 << 20, uploaded: 10 << 20, elapsed: uploadProgressMinInterval - time.Millisecond, want: false},
		{name: "maximum time threshold", total: 100 << 20, uploaded: 1 << 20, elapsed: uploadProgressMaxInterval, want: true},
		{name: "below thresholds", total: 100 << 20, uploaded: 1 << 20, elapsed: uploadProgressMaxInterval - time.Millisecond, want: false},
		{name: "completion", total: 100, uploaded: 100, lastPercent: 99, elapsed: uploadProgressMinInterval, want: true},
		{name: "completion already reported", total: 100, uploaded: 100, lastPercent: 100, elapsed: uploadProgressMinInterval, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldUpdateUploadProgress(tt.total, tt.uploaded, tt.lastPercent, tt.elapsed); got != tt.want {
				t.Fatalf("shouldUpdateUploadProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func (*uploadProgressRecorder) OnStart(context.Context, TaskInfo)       {}
func (*uploadProgressRecorder) OnProgress(context.Context, TaskInfo)    {}
func (*uploadProgressRecorder) OnDone(context.Context, TaskInfo, error) {}

func (r *uploadProgressRecorder) OnUploadStart(_ context.Context, _ TaskInfo, total int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.starts++
	r.startTotal = total
}

func (r *uploadProgressRecorder) OnUploadProgress(_ context.Context, _ TaskInfo, uploaded, total int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uploaded = uploaded
	r.total = total
}

func TestUploadProgressAggregatesItemsAndHandlesRetry(t *testing.T) {
	recorder := new(uploadProgressRecorder)
	task := &Task{Progress: recorder, totalSize: 300, uploaded: make(map[string]int64)}
	ctx := context.Background()
	first := task.uploadCallback(ctx, "first")
	second := task.uploadCallback(ctx, "second")

	first(40, 100)
	second(50, 200)
	first(80, 100)
	second(10, 200) // A retry may restart one item from zero.
	first(150, 100) // Per-item progress is clamped to its declared total.

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.starts != 1 {
		t.Fatalf("upload starts = %d, want 1", recorder.starts)
	}
	if recorder.startTotal != 300 {
		t.Fatalf("start total = %d, want 300", recorder.startTotal)
	}
	if recorder.uploaded != 110 || recorder.total != 300 {
		t.Fatalf("aggregate progress = %d/%d, want 110/300", recorder.uploaded, recorder.total)
	}
}

func TestUploadProgressConcurrentItems(t *testing.T) {
	recorder := new(uploadProgressRecorder)
	task := &Task{Progress: recorder, totalSize: 200, uploaded: make(map[string]int64)}
	ctx := context.Background()

	var wg sync.WaitGroup
	for _, id := range []string{"first", "second"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			callback := task.uploadCallback(ctx, id)
			for uploaded := int64(1); uploaded <= 100; uploaded++ {
				callback(uploaded, 100)
			}
		}()
	}
	wg.Wait()

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.starts != 1 {
		t.Fatalf("upload starts = %d, want 1", recorder.starts)
	}
	if recorder.uploaded != 200 || recorder.total != 200 {
		t.Fatalf("aggregate progress = %d/%d, want 200/200", recorder.uploaded, recorder.total)
	}
}
