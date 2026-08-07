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
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

type uploadProgressRecorder struct {
	mu            sync.Mutex
	starts        int
	startTotal    int64
	uploaded      int64
	total         int64
	notifications []int64
}

type orderedUploadProgressRecorder struct {
	firstEntered  chan struct{}
	releaseFirst  chan struct{}
	secondEntered chan struct{}
	mu            sync.Mutex
	notifications []int64
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
	r.notifications = append(r.notifications, uploaded)
}

func (*orderedUploadProgressRecorder) OnStart(context.Context, TaskInfo)       {}
func (*orderedUploadProgressRecorder) OnProgress(context.Context, TaskInfo)    {}
func (*orderedUploadProgressRecorder) OnDone(context.Context, TaskInfo, error) {}
func (*orderedUploadProgressRecorder) OnUploadStart(context.Context, TaskInfo, int64) {
}

func (r *orderedUploadProgressRecorder) OnUploadProgress(_ context.Context, _ TaskInfo, uploaded, _ int64) {
	if uploaded == 100 {
		close(r.firstEntered)
		<-r.releaseFirst
	}
	if uploaded == 200 {
		close(r.secondEntered)
	}
	r.mu.Lock()
	r.notifications = append(r.notifications, uploaded)
	r.mu.Unlock()
}

func TestDownloadProgressContinuesAfterUploadStarts(t *testing.T) {
	task := &Task{
		ID:         "interleaved",
		elems:      make([]TaskElement, 2),
		totalSize:  20 << 20,
		processing: make(map[string]TaskElementInfo),
	}
	progress := new(Progress)
	progress.OnStart(t.Context(), task)
	progress.OnUploadStart(t.Context(), task, task.totalSize)
	task.downloaded.Store(10 << 20)
	progress.OnProgress(t.Context(), task)

	if got := progress.downloadLastUpdatePercent.Load(); got != 50 {
		t.Fatalf("download progress after upload start = %d%%, want 50%%", got)
	}
}

func TestUploadProgressAggregatesItemsAndHandlesRetry(t *testing.T) {
	recorder := new(uploadProgressRecorder)
	task := &Task{Progress: recorder, totalSize: 300, uploaded: make(map[string]int64)}
	task.uploadTotalSize.Store(300)
	ctx := context.Background()
	first := task.uploadCallback(ctx, "first")
	second := task.uploadCallback(ctx, "second")

	first(40, 100)
	second(50, 200)
	first(80, 100)
	second(10, 200) // A retry may restart one item from zero; aggregate progress must not regress.
	first(150, 100) // Per-item progress is clamped to its declared total.

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.starts != 1 {
		t.Fatalf("upload starts = %d, want 1", recorder.starts)
	}
	if recorder.startTotal != 300 {
		t.Fatalf("start total = %d, want 300", recorder.startTotal)
	}
	if recorder.uploaded != 150 || recorder.total != 300 {
		t.Fatalf("aggregate progress = %d/%d, want 150/300", recorder.uploaded, recorder.total)
	}
	for i := 1; i < len(recorder.notifications); i++ {
		if recorder.notifications[i] < recorder.notifications[i-1] {
			t.Fatalf("aggregate progress regressed: %v", recorder.notifications)
		}
	}
}

func TestRecordedUploadSizesExcludeStreamedFiles(t *testing.T) {
	recorder := new(uploadProgressRecorder)
	saver := new(fallbackBatchSaver)
	files := []TaskElement{
		{
			ID:      "streamed",
			Storage: saver,
			File:    tfile.NewTGFile(nil, nil, 100, "streamed.bin"),
			stream:  true,
		},
		{
			ID:      "cached",
			Storage: saver,
			File:    tfile.NewTGFile(nil, nil, 200, "cached.bin"),
		},
		{
			ID:             "batch-cached",
			Storage:        saver,
			File:           tfile.NewTGFile(nil, nil, 50, "batch-cached.bin"),
			stream:         true,
			sourceGroupKey: "album",
		},
	}
	task := NewBatchTGFileTask("mixed", t.Context(), files, recorder, true)
	if task.totalSize != 350 {
		t.Fatalf("download total = %d, want 350", task.totalSize)
	}
	task.recordDownloadComplete(0)
	task.recordDownloadComplete(200)
	task.recordDownloadComplete(50)
	if got := task.uploadTotalSize.Load(); got != 250 {
		t.Fatalf("tracked upload total = %d, want 250", got)
	}
	task.uploadCallback(t.Context(), "cached")(200, 200)
	task.uploadCallback(t.Context(), "batch-cached")(50, 50)

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.startTotal != 250 {
		t.Fatalf("upload start total = %d, want 250", recorder.startTotal)
	}
	if recorder.uploaded != 250 || recorder.total != 250 {
		t.Fatalf("aggregate upload progress = %d/%d, want 250/250", recorder.uploaded, recorder.total)
	}
}

func TestZeroMetadataSizeUsesActualUploadSize(t *testing.T) {
	recorder := new(uploadProgressRecorder)
	files := []TaskElement{{
		ID:   "photo",
		File: tfile.NewTGFile(nil, nil, 0, "photo.jpg"),
	}}
	task := NewBatchTGFileTask("photo", t.Context(), files, recorder, true)
	if task.totalSize != 0 {
		t.Fatalf("metadata download total = %d, want 0", task.totalSize)
	}

	task.recordDownloadComplete(25)
	task.uploadCallback(t.Context(), "photo")(25, 25)

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.startTotal != 25 {
		t.Fatalf("upload start total = %d, want 25", recorder.startTotal)
	}
	if recorder.uploaded != 25 || recorder.total != 25 {
		t.Fatalf("photo upload progress = %d/%d, want 25/25", recorder.uploaded, recorder.total)
	}
}

func TestEqualCompletionCallbackCanReportRateLimitedUpload(t *testing.T) {
	progress := new(Progress)
	task := &Task{
		ID:         "completion",
		elems:      []TaskElement{{ID: "file"}},
		Progress:   progress,
		totalSize:  100,
		processing: make(map[string]TaskElementInfo),
		uploaded:   make(map[string]int64),
	}
	task.downloaded.Store(task.totalSize)
	task.recordDownloadComplete(100)
	progress.OnStart(t.Context(), task)
	onProgress := task.uploadCallback(t.Context(), "file")
	onProgress(100, 100)
	if got := progress.uploadLastUpdatePercent.Load(); got != 0 {
		t.Fatalf("rate-limited completion = %d%%, want 0%%", got)
	}

	progress.uploadLastUpdateAt.Store(time.Now().Add(-uploadProgressMinInterval).UnixNano())
	onProgress(100, 100)
	if got := progress.uploadLastUpdatePercent.Load(); got != 100 {
		t.Fatalf("repeated completion callback = %d%%, want 100%%", got)
	}
}

func TestBatchProgressSwitchesToUploadOnlyAfterDownloadsFinish(t *testing.T) {
	task := &Task{
		ID:         "interleaved",
		elems:      make([]TaskElement, 2),
		totalSize:  20 << 20,
		processing: make(map[string]TaskElementInfo),
	}
	progress := new(Progress)
	progress.OnStart(t.Context(), task)
	task.downloaded.Store(10 << 20)
	progress.OnProgress(t.Context(), task)
	progress.OnUploadStart(t.Context(), task, task.totalSize)
	progress.uploadLastUpdateAt.Store(time.Now().Add(-uploadProgressMinInterval).UnixNano())
	progress.OnUploadProgress(t.Context(), task, 5<<20, task.totalSize)

	if progress.uploadVisible.Load() {
		t.Fatal("upload phase became visible before all downloads finished")
	}
	if got := progress.downloadLastUpdatePercent.Load(); got != 50 {
		t.Fatalf("download progress before upload phase = %d%%, want 50%%", got)
	}

	task.downloaded.Store(task.totalSize)
	progress.OnUploadProgress(t.Context(), task, 10<<20, task.totalSize)
	if progress.uploadVisible.Load() {
		t.Fatal("upload phase became visible from byte totals before downloads completed")
	}
	task.recordDownloadComplete(10 << 20)
	task.recordDownloadComplete(10 << 20)
	progress.OnUploadProgress(t.Context(), task, 10<<20, task.totalSize)
	if !progress.uploadVisible.Load() {
		t.Fatal("upload phase did not become visible after all downloads finished")
	}
	if got := progress.uploadLastUpdatePercent.Load(); got != 50 {
		t.Fatalf("upload progress after phase switch = %d%%, want 50%%", got)
	}

	progress.OnProgress(t.Context(), task)
	if got := progress.downloadLastUpdatePercent.Load(); got != 50 {
		t.Fatalf("delayed download update rewrote upload phase: download progress = %d%%, want 50%%", got)
	}
}

func TestUploadProgressConcurrentItems(t *testing.T) {
	recorder := new(uploadProgressRecorder)
	task := &Task{Progress: recorder, totalSize: 200, uploaded: make(map[string]int64)}
	task.uploadTotalSize.Store(200)
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

func TestUploadProgressNotificationsRemainOrdered(t *testing.T) {
	recorder := &orderedUploadProgressRecorder{
		firstEntered:  make(chan struct{}),
		releaseFirst:  make(chan struct{}),
		secondEntered: make(chan struct{}),
	}
	task := &Task{Progress: recorder, totalSize: 200, uploaded: make(map[string]int64)}
	task.uploadTotalSize.Store(200)
	first := task.uploadCallback(t.Context(), "first")
	second := task.uploadCallback(t.Context(), "second")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		first(100, 100)
	}()
	<-recorder.firstEntered

	wg.Add(1)
	go func() {
		defer wg.Done()
		second(100, 100)
	}()

	overtook := false
	select {
	case <-recorder.secondEntered:
		overtook = true
	case <-time.After(100 * time.Millisecond):
	}
	close(recorder.releaseFirst)
	wg.Wait()

	if overtook {
		t.Fatal("later aggregate notification overtook an earlier callback")
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if len(recorder.notifications) != 2 || recorder.notifications[0] != 100 || recorder.notifications[1] != 200 {
		t.Fatalf("upload notifications = %v, want [100 200]", recorder.notifications)
	}
}

func TestDownloadCompletionCannotChangeUploadSnapshot(t *testing.T) {
	recorder := &orderedUploadProgressRecorder{
		firstEntered:  make(chan struct{}),
		releaseFirst:  make(chan struct{}),
		secondEntered: make(chan struct{}),
	}
	task := &Task{
		elems:    make([]TaskElement, 2),
		Progress: recorder,
		uploaded: make(map[string]int64),
	}
	task.recordDownloadComplete(100)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		task.uploadCallback(t.Context(), "first")(100, 100)
	}()
	<-recorder.firstEntered

	recordStarted := make(chan struct{})
	recordDone := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(recordStarted)
		task.recordDownloadComplete(100)
		close(recordDone)
	}()
	<-recordStarted

	completedDuringCallback := false
	select {
	case <-recordDone:
		completedDuringCallback = true
	case <-time.After(100 * time.Millisecond):
	}
	close(recorder.releaseFirst)
	wg.Wait()
	if completedDuringCallback {
		t.Fatal("download completion changed upload state while a progress snapshot was being reported")
	}

	task.uploadCallback(t.Context(), "second")(100, 100)
	select {
	case <-recorder.secondEntered:
	default:
		t.Fatal("final upload progress did not use the completed 200-byte total")
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
	task.uploadTotalSize.Store(6)
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
