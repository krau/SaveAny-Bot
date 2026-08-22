package batchtfile

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

type progressRegressionRecorder struct {
	mu            sync.Mutex
	startTotal    int64
	notifications []int64
}

func (*progressRegressionRecorder) OnStart(context.Context, TaskInfo)       {}
func (*progressRegressionRecorder) OnProgress(context.Context, TaskInfo)    {}
func (*progressRegressionRecorder) OnDone(context.Context, TaskInfo, error) {}

func (r *progressRegressionRecorder) OnUploadStart(_ context.Context, _ TaskInfo, total int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.startTotal = total
}

func (r *progressRegressionRecorder) OnUploadProgress(_ context.Context, _ TaskInfo, uploaded, _ int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifications = append(r.notifications, uploaded)
}

type orderedProgressRegressionRecorder struct {
	firstEntered  chan struct{}
	releaseFirst  chan struct{}
	secondEntered chan struct{}
	mu            sync.Mutex
	notifications []int64
}

func (*orderedProgressRegressionRecorder) OnStart(context.Context, TaskInfo)       {}
func (*orderedProgressRegressionRecorder) OnProgress(context.Context, TaskInfo)    {}
func (*orderedProgressRegressionRecorder) OnDone(context.Context, TaskInfo, error) {}
func (*orderedProgressRegressionRecorder) OnUploadStart(context.Context, TaskInfo, int64) {
}

func (r *orderedProgressRegressionRecorder) OnUploadProgress(_ context.Context, _ TaskInfo, uploaded, _ int64) {
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

func TestBatchProgressShowsTransferSpeedAndSize(t *testing.T) {
	useProgressRegressionLocale(t)
	task := newProgressRegressionTask(nil,
		progressRegressionFile{"downloading", 1000},
		progressRegressionFile{"uploading", 1000},
		progressRegressionFile{"waiting", 1000},
	)
	started := time.Unix(100, 0)
	task.markItemActive("downloading", false, started)
	task.recordItemDownload("downloading", 500, started.Add(time.Second))
	task.recordItemDownloaded("uploading", 1000)
	task.recordItemUpload("uploading", 0, 1000, started.Add(time.Second))
	task.recordItemUpload("uploading", 250, 1000, started.Add(2*time.Second))

	message := buildBatchProgressMessage(task, nil, 2)
	if message.Err != nil {
		t.Fatalf("buildBatchProgressMessage() failed: %v", message.Err)
	}
	assertProgressRegressionContains(t, message.Text,
		"状态：✅ 0 ｜ 📥 0 ｜ ⏳ 1",
		"⬇️ 1/3 下载中",
		"速度：500 B/s",
		"大小：500 B / 1000 B",
		"⬆️ 2/3 上传中",
		"速度：250 B/s",
		"大小：250 B / 1000 B",
	)
	bold, _, blockquote, _ := batchEntityCounts(message.Entities)
	if bold != 3 || blockquote != 2 {
		t.Fatalf("entity counts = bold:%d blockquote:%d, want bold:3 blockquote:2", bold, blockquote)
	}
}

func TestBatchProgressHeaderShowsTotalSize(t *testing.T) {
	useProgressRegressionLocale(t)
	task := newProgressRegressionTask(nil,
		progressRegressionFile{"first", 1024},
		progressRegressionFile{"second", 1024},
	)
	message := buildBatchProgressMessage(task, nil, 2)
	if message.Err != nil {
		t.Fatalf("buildBatchProgressMessage() failed: %v", message.Err)
	}
	assertProgressRegressionContains(t, message.Text,
		"文件：2 ｜ 总大小：2.00 KB",
	)

	i18n.Init("en")
	english := buildBatchProgressMessage(task, nil, 2)
	if english.Err != nil {
		t.Fatalf("English batch template failed: %v", english.Err)
	}
	assertProgressRegressionContains(t, english.Text,
		"Files: 2 | Total size: 2.00 KB",
	)
}

func TestBatchProgressLimitsRowsWithoutHidingActiveUpload(t *testing.T) {
	useProgressRegressionLocale(t)
	task := newProgressRegressionTask(nil,
		progressRegressionFile{"confirm-01", 100},
		progressRegressionFile{"confirm-02", 100},
		progressRegressionFile{"uploading", 100},
		progressRegressionFile{"downloading", 100},
	)
	started := time.Unix(100, 0)
	task.recordItemUpload("confirm-01", 100, 100, started)
	task.recordItemUpload("confirm-02", 100, 100, started)
	task.recordItemUpload("uploading", 40, 100, started.Add(time.Second))
	task.markItemActive("downloading", false, started)

	message := buildBatchProgressText(task, nil, 2)
	assertProgressRegressionContains(t, message,
		"uploading.bin",
		"downloading.bin",
		"☁️ 已上传，等待整组发送：2",
	)
	if strings.Contains(message, "confirm-01.bin") || strings.Contains(message, "confirm-02.bin") {
		t.Fatalf("confirmation rows displaced active transfers:\n%s", message)
	}
}

func TestBatchProgressTemplateOwnsStylesAndEscapesValues(t *testing.T) {
	useProgressRegressionLocale(t)
	fileID := `<b>A&B</b>`
	task := newProgressRegressionTask(nil, progressRegressionFile{fileID, 100})
	task.markItemRetry(fileID, FailureStageUpload, 1, 3, errors.New(`<i>remote & failed</i>`))

	message := buildBatchProgressMessage(task, nil, 1)
	if message.Err != nil {
		t.Fatalf("buildBatchProgressMessage() failed: %v", message.Err)
	}
	assertProgressRegressionContains(t, message.Text,
		`<b>A&B</b>.bin`,
		`<i>remote & failed</i>`,
	)
	bold, _, blockquote, italic := batchEntityCounts(message.Entities)
	if bold != 2 || blockquote != 1 || italic != 0 {
		t.Fatalf("entity counts = bold:%d blockquote:%d italic:%d", bold, blockquote, italic)
	}

	i18n.Init("en")
	english := buildBatchProgressMessage(task, nil, 1)
	if english.Err != nil {
		t.Fatalf("English batch template failed: %v", english.Err)
	}
	assertProgressRegressionContains(t, english.Text, "📦 Processing", "Retrying upload", `<b>A&B</b>.bin`)
}

func TestDownloadProgressContinuesAfterUploadStarts(t *testing.T) {
	useProgressRegressionLocale(t)
	progress := new(Progress)
	task := newProgressRegressionTask(progress,
		progressRegressionFile{"uploading", 100},
		progressRegressionFile{"downloading", 100},
	)
	progress.OnStart(t.Context(), task)
	task.recordDownloadComplete("uploading", 100)
	task.uploadCallback(t.Context(), "uploading")(50, 100)

	started := time.Unix(100, 0)
	task.markItemActive("downloading", false, started)
	task.recordItemDownload("downloading", 50, started.Add(time.Second))
	progress.updateMu.Lock()
	progress.lastUpdateAt = time.Now().Add(-progressRenderInterval)
	progress.updateMu.Unlock()
	progress.OnProgress(t.Context(), task)

	progress.updateMu.Lock()
	text := progress.lastText
	progress.updateMu.Unlock()
	assertProgressRegressionContains(t, text,
		"uploading.bin",
		"🟩🟩🟩🟩🟩⬜️⬜️⬜️⬜️⬜️ 50%",
		"总速度：⬇️ 50 B/s ｜ ⬆️ 0 B/s",
		"🔄 另有 1 个文件正在处理",
	)
}

func TestBatchUploadIgnoresOutOfOrderBytesAndAllowsRetryReset(t *testing.T) {
	recorder := new(progressRegressionRecorder)
	task := newProgressRegressionTask(recorder, progressRegressionFile{"file", 100})
	task.recordDownloadComplete("file", 100)
	callback := task.uploadCallback(t.Context(), "file")
	callback(80, 100)
	callback(10, 100)

	if got := task.Items()[0].Uploaded; got != 80 {
		t.Fatalf("out-of-order callback regressed item to %d, want 80", got)
	}
	task.markItemRetry("file", FailureStageUpload, 1, 3, context.DeadlineExceeded)
	callback(0, 100)
	callback(10, 100)
	if got := task.Items()[0].Uploaded; got != 10 {
		t.Fatalf("retry did not reset item progress: got %d, want 10", got)
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	for index := 1; index < len(recorder.notifications); index++ {
		if recorder.notifications[index] < recorder.notifications[index-1] {
			t.Fatalf("aggregate progress regressed: %v", recorder.notifications)
		}
	}
}

func TestUploadProgressNotificationsRemainOrdered(t *testing.T) {
	recorder := &orderedProgressRegressionRecorder{
		firstEntered:  make(chan struct{}),
		releaseFirst:  make(chan struct{}),
		secondEntered: make(chan struct{}),
	}
	task := newProgressRegressionTask(recorder,
		progressRegressionFile{"first", 100},
		progressRegressionFile{"second", 100},
	)
	task.recordDownloadComplete("first", 100)
	task.recordDownloadComplete("second", 100)
	first := task.uploadCallback(t.Context(), "first")
	second := task.uploadCallback(t.Context(), "second")

	var wait sync.WaitGroup
	wait.Go(func() {
		first(100, 100)
	})
	<-recorder.firstEntered
	wait.Go(func() {
		second(100, 100)
	})

	overtook := false
	select {
	case <-recorder.secondEntered:
		overtook = true
	case <-time.After(100 * time.Millisecond):
	}
	close(recorder.releaseFirst)
	wait.Wait()
	if overtook {
		t.Fatal("later aggregate notification overtook the first callback")
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if got := recorder.notifications; len(got) != 2 || got[0] != 100 || got[1] != 200 {
		t.Fatalf("upload notifications = %v, want [100 200]", got)
	}
}

func TestBatchUploadUsesActualSizeWhenMetadataIsUnknown(t *testing.T) {
	recorder := new(progressRegressionRecorder)
	task := newProgressRegressionTask(recorder, progressRegressionFile{"photo", 0})
	task.recordDownloadComplete("photo", 25)
	task.uploadCallback(t.Context(), "photo")(25, 25)

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.startTotal != 25 {
		t.Fatalf("upload start total = %d, want actual size 25", recorder.startTotal)
	}
	if got := task.ActualTotalSize(); got != 25 {
		t.Fatalf("actual total size = %d, want 25", got)
	}
}

type progressRegressionFile struct {
	id   string
	size int64
}

func newProgressRegressionTask(progress ProgressTracker, files ...progressRegressionFile) *Task {
	elems := make([]TaskElement, 0, len(files))
	for _, file := range files {
		elems = append(elems, TaskElement{
			ID:   file.id,
			File: tfile.NewTGFile(nil, nil, file.size, file.id+".bin"),
		})
	}
	return NewBatchTGFileTask("progress-regression", context.Background(), elems, progress, true)
}

func useProgressRegressionLocale(t *testing.T) {
	t.Helper()
	i18n.Init("zh-Hans")
	t.Cleanup(func() { i18n.Init("zh-Hans") })
}

func assertProgressRegressionContains(t *testing.T, value string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(value, want) {
			t.Fatalf("text does not contain %q:\n%s", want, value)
		}
	}
}

func batchEntityCounts(entities []tg.MessageEntityClass) (bold, code, blockquote, italic int) {
	for _, messageEntity := range entities {
		switch messageEntity.(type) {
		case *tg.MessageEntityBold:
			bold++
		case *tg.MessageEntityCode:
			code++
		case *tg.MessageEntityBlockquote:
			blockquote++
		case *tg.MessageEntityItalic:
			italic++
		}
	}
	return
}
