package batchtfile

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/krau/SaveAny-Bot/common/i18n"
	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/storage"
)

type uploadProgressRecorder struct {
	mu         sync.Mutex
	starts     int
	startTotal int64
	uploaded   int64
	total      int64
}

type fallbackBatchSaver struct {
	saveBatchCalled bool
}

func (*fallbackBatchSaver) Init(context.Context, storcfg.StorageConfig) error { return nil }
func (*fallbackBatchSaver) Type() storenum.StorageType                        { return storenum.Local }
func (*fallbackBatchSaver) Name() string                                      { return "test" }
func (*fallbackBatchSaver) Save(context.Context, io.Reader, string) error     { return nil }
func (*fallbackBatchSaver) Exists(context.Context, string) bool               { return false }

func (s *fallbackBatchSaver) SaveBatch(_ context.Context, items []storagetypes.BatchItem) error {
	s.saveBatchCalled = true
	for _, item := range items {
		if _, err := io.Copy(io.Discard, item.Reader); err != nil {
			return err
		}
	}
	return nil
}

type nativeBatchSaver struct {
	fallbackBatchSaver
	saveBatchWithProgressCalled bool
}

func (s *nativeBatchSaver) SaveBatchWithProgress(
	_ context.Context,
	items []storagetypes.BatchItem,
	onProgress func(index int, uploaded, total int64),
) error {
	s.saveBatchWithProgressCalled = true
	onProgress(-1, 1, 1)
	onProgress(len(items), 1, 1)
	for index, item := range items {
		onProgress(index, item.Size/2, item.Size)
		onProgress(index, item.Size, item.Size)
	}
	return nil
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

func TestBatchProgressSummaryLocalized(t *testing.T) {
	t.Cleanup(func() { i18n.Init("zh-Hans") })
	tests := []struct {
		lang string
		want string
	}{
		{lang: "en", want: "3.00 MB (2 files)"},
		{lang: "zh-Hans", want: "3.00 MB (2 个文件)"},
	}
	for _, tt := range tests {
		i18n.Init(tt.lang)
		if got := batchProgressSummary(3*1024*1024, 2); got != tt.want {
			t.Fatalf("batchProgressSummary() with %s locale = %q, want %q", tt.lang, got, tt.want)
		}
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

func TestSaveBatchItemsUsesNativeProgress(t *testing.T) {
	saver := new(nativeBatchSaver)
	recorder := new(uploadProgressRecorder)
	task, group, items := batchUploadFixture(recorder, saver)

	if err := task.saveBatchItems(context.Background(), group, items); err != nil {
		t.Fatalf("saveBatchItems() failed: %v", err)
	}
	if !saver.saveBatchWithProgressCalled {
		t.Fatal("SaveBatchWithProgress() was not called")
	}
	if saver.saveBatchCalled {
		t.Fatal("SaveBatch() fallback was called for a native progress saver")
	}
	assertRecordedBatchUpload(t, recorder)
}

func TestSaveBatchItemsFallsBackToTrackedReaders(t *testing.T) {
	saver := new(fallbackBatchSaver)
	recorder := new(uploadProgressRecorder)
	task, group, items := batchUploadFixture(recorder, saver)

	if err := task.saveBatchItems(context.Background(), group, items); err != nil {
		t.Fatalf("saveBatchItems() failed: %v", err)
	}
	if !saver.saveBatchCalled {
		t.Fatal("SaveBatch() fallback was not called")
	}
	assertRecordedBatchUpload(t, recorder)
}

func batchUploadFixture(
	recorder *uploadProgressRecorder,
	saver storage.StorageBatchSaver,
) (*Task, executionGroup, []storagetypes.BatchItem) {
	elems := []*TaskElement{{ID: "first"}, {ID: "second"}}
	task := &Task{
		Progress:  recorder,
		totalSize: 6,
		uploaded:  make(map[string]int64),
	}
	group := executionGroup{elems: elems, batchSaver: saver}
	items := []storagetypes.BatchItem{
		{Reader: bytes.NewReader([]byte("abc")), Size: 3},
		{Reader: bytes.NewReader([]byte("def")), Size: 3},
	}
	return task, group, items
}

func assertRecordedBatchUpload(t *testing.T, recorder *uploadProgressRecorder) {
	t.Helper()
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.starts != 1 {
		t.Fatalf("upload starts = %d, want 1", recorder.starts)
	}
	if recorder.uploaded != 6 || recorder.total != 6 {
		t.Fatalf("aggregate progress = %d/%d, want 6/6", recorder.uploaded, recorder.total)
	}
}
