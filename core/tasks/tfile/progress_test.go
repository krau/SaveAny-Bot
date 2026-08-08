package tfile

import (
	"context"
	"sync"
	"testing"
	"time"
)

type progressTestTaskInfo struct{}

func (progressTestTaskInfo) TaskID() string      { return "task" }
func (progressTestTaskInfo) FileName() string    { return "file.bin" }
func (progressTestTaskInfo) FileSize() int64     { return 100 << 20 }
func (progressTestTaskInfo) StoragePath() string { return "file.bin" }
func (progressTestTaskInfo) StorageName() string { return "test" }

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
		{name: "completion rate limited", total: 100, uploaded: 100, lastPercent: 99, elapsed: uploadProgressMinInterval - time.Millisecond, want: false},
		{name: "completion already reported", total: 100, uploaded: 100, lastPercent: 100, elapsed: uploadProgressMinInterval, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldUpdateUploadProgress(tt.total, tt.uploaded, tt.lastPercent, tt.elapsed)
			if got != tt.want {
				t.Fatalf("shouldUpdateUploadProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadProgressConcurrentCallbacks(t *testing.T) {
	progress := new(Progress)
	ctx := context.Background()
	info := progressTestTaskInfo{}
	const total = int64(100 << 20)

	progress.OnUploadStart(ctx, info, total)
	progress.lastUpdateAt.Store(time.Now().Add(-uploadProgressMaxInterval).UnixNano())

	var wg sync.WaitGroup
	for uploaded := int64(1 << 20); uploaded <= total; uploaded += 1 << 20 {
		uploaded := uploaded
		wg.Add(1)
		go func() {
			defer wg.Done()
			progress.OnUploadProgress(ctx, info, uploaded, total)
		}()
	}
	wg.Wait()

	percent := progress.lastUpdatePercent.Load()
	if percent <= 0 || percent > 100 {
		t.Fatalf("last upload percentage = %d, want a value in (0, 100]", percent)
	}
}
