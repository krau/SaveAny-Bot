package batchtfile

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

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

type itemUploadSnapshotRecorder struct {
	mu           sync.Mutex
	aggregates   []int64
	itemUploaded []int64
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

func (*itemUploadSnapshotRecorder) OnStart(context.Context, TaskInfo)       {}
func (*itemUploadSnapshotRecorder) OnProgress(context.Context, TaskInfo)    {}
func (*itemUploadSnapshotRecorder) OnDone(context.Context, TaskInfo, error) {}
func (*itemUploadSnapshotRecorder) OnUploadStart(context.Context, TaskInfo, int64) {
}

func (r *itemUploadSnapshotRecorder) OnUploadProgress(_ context.Context, info TaskInfo, uploaded, _ int64) {
	items := info.Items()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.aggregates = append(r.aggregates, uploaded)
	if len(items) > 0 {
		r.itemUploaded = append(r.itemUploaded, items[0].Uploaded)
	}
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

func TestUploadProgressAggregatesItemsAndIgnoresOutOfOrderCallbacks(t *testing.T) {
	recorder := new(uploadProgressRecorder)
	task := &Task{Progress: recorder, totalSize: 300, uploaded: make(map[string]int64)}
	task.uploadTotalSize.Store(300)
	ctx := context.Background()
	first := task.uploadCallback(ctx, "first")
	second := task.uploadCallback(ctx, "second")

	first(40, 100)
	second(50, 200)
	first(80, 100)
	second(10, 200) // An out-of-order callback must not regress aggregate progress.
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

func TestOutOfOrderUploadCallbackDoesNotRegressItemSnapshot(t *testing.T) {
	recorder := new(itemUploadSnapshotRecorder)
	task := NewBatchTGFileTask("retry", t.Context(), []TaskElement{{
		ID:   "file",
		File: tfile.NewTGFile(nil, nil, 100, "file.bin"),
	}}, recorder, true)
	task.recordDownloadComplete("file", 100)
	onProgress := task.uploadCallback(t.Context(), "file")
	onProgress(80, 100)
	onProgress(10, 100)

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if got := recorder.aggregates; len(got) != 2 || got[0] != 80 || got[1] != 80 {
		t.Fatalf("aggregate snapshots = %v, want [80 80]", got)
	}
	if got := recorder.itemUploaded; len(got) != 2 || got[0] != 80 || got[1] != 80 {
		t.Fatalf("item snapshots = %v, want [80 80]", got)
	}
}

func TestExplicitRetryCanResetItemWithoutRegressingAggregate(t *testing.T) {
	recorder := new(itemUploadSnapshotRecorder)
	task := NewBatchTGFileTask("retry", t.Context(), []TaskElement{{
		ID:   "file",
		File: tfile.NewTGFile(nil, nil, 100, "file.bin"),
	}}, recorder, true)
	task.recordDownloadComplete("file", 100)
	onProgress := task.uploadCallback(t.Context(), "file")
	onProgress(80, 100)
	task.markItemRetry("file", FailureStageUpload, 1, 3, errors.New("retry"))
	onProgress(0, 100)
	onProgress(10, 100)

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if got := recorder.aggregates; len(got) != 3 || got[0] != 80 || got[1] != 80 || got[2] != 80 {
		t.Fatalf("aggregate snapshots = %v, want [80 80 80]", got)
	}
	if got := recorder.itemUploaded; len(got) != 3 || got[0] != 80 || got[1] != 0 || got[2] != 10 {
		t.Fatalf("item snapshots = %v, want [80 0 10]", got)
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
	task.recordDownloadComplete("streamed", 0)
	task.recordDownloadComplete("cached", 200)
	task.recordDownloadComplete("batch-cached", 50)
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

	task.recordDownloadComplete("photo", 25)
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
	task.recordDownloadComplete("first", 100)

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
		task.recordDownloadComplete("second", 100)
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
