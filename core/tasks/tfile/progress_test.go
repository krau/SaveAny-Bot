package tfile

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
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
		{name: "out of order callback", total: 100, uploaded: 40, lastPercent: 60, elapsed: uploadProgressMaxInterval, want: false},
		{name: "same percentage refresh", total: 1000, uploaded: 601, lastPercent: 60, elapsed: uploadProgressMaxInterval, want: true},
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

func TestShouldUpdateUnknownSizeDownloadProgress(t *testing.T) {
	if shouldUpdateSingleDownloadProgress(0, 1024, 0, uploadProgressMaxInterval-time.Millisecond) {
		t.Fatal("unknown-size download updated before the time threshold")
	}
	if !shouldUpdateSingleDownloadProgress(0, 1024, 0, uploadProgressMaxInterval) {
		t.Fatal("unknown-size download did not update at the time threshold")
	}
	if shouldUpdateSingleDownloadProgress(0, 0, 0, uploadProgressMaxInterval) {
		t.Fatal("unknown-size download updated without downloaded bytes")
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
	if progress.uploadedBytes != total {
		t.Fatalf("maximum uploaded bytes = %d, want %d", progress.uploadedBytes, total)
	}
}

func TestSingleUploadProgressDoesNotRegress(t *testing.T) {
	progress := new(Progress)
	ctx := context.Background()
	info := progressTestTaskInfo{}
	const total = int64(100 << 20)

	progress.OnUploadStart(ctx, info, total)
	progress.lastUpdateAt.Store(time.Now().Add(-uploadProgressMaxInterval).UnixNano())
	progress.OnUploadProgress(ctx, info, 80<<20, total)
	progress.lastUpdateAt.Store(time.Now().Add(-uploadProgressMaxInterval).UnixNano())
	progress.OnUploadProgress(ctx, info, 40<<20, total)

	if progress.uploadedBytes != 80<<20 {
		t.Fatalf("uploaded bytes regressed to %d", progress.uploadedBytes)
	}
	if progress.lastUpdatePercent.Load() != 80 {
		t.Fatalf("rendered upload percentage regressed to %d", progress.lastUpdatePercent.Load())
	}
}

func TestSingleDownloadProgressUsesRichLayout(t *testing.T) {
	useSingleProgressTestLocale(t, "zh-Hans")
	message := buildSingleProgressMessage(
		progressTestTaskInfo{},
		singlePhaseDownloading,
		50<<20,
		100<<20,
		10<<20,
		0,
	)
	assertSingleContainsAll(t, message.Text,
		"📦 正在处理",
		"⬇️ 下载中",
		"file.bin",
		"🟩🟩🟩🟩🟩⬜️⬜️⬜️⬜️⬜️ 50%",
		"速度：10.00 MB/s",
		"大小：50.00 MB / 100.00 MB",
		"保存至：[test]:file.bin",
	)
	assertSingleRichEntities(t, message, 1)
}

func TestSingleUploadAndRetryProgressUseRichLayout(t *testing.T) {
	useSingleProgressTestLocale(t, "zh-Hans")
	upload := buildSingleProgressMessage(
		progressTestTaskInfo{},
		singlePhaseUploading,
		25<<20,
		100<<20,
		5<<20,
		1,
	)
	assertSingleContainsAll(t, upload.Text,
		"⬆️ 上传中",
		"🟩🟩⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️ 25%",
		"速度：5.00 MB/s",
		"大小：25.00 MB / 100.00 MB",
	)

	retrying := buildSingleProgressMessage(
		progressTestTaskInfo{},
		singleUploadPhase(2),
		25<<20,
		100<<20,
		5<<20,
		2,
	)
	assertSingleContainsAll(t, retrying.Text,
		"🔁 上传重试",
		"尝试次数：2",
		"速度：5.00 MB/s",
		"大小：25.00 MB / 100.00 MB",
	)
	assertSingleRichEntities(t, upload, 1)
	assertSingleRichEntities(t, retrying, 1)
	if phase := singleUploadPhase(2); phase != singlePhaseRetrying {
		t.Fatalf("second upload attempt phase = %v, want retrying", phase)
	}
}

func TestSingleProgressUnknownDownloadSize(t *testing.T) {
	useSingleProgressTestLocale(t, "zh-Hans")
	message := buildSingleProgressMessage(
		progressTestTaskInfo{},
		singlePhaseDownloading,
		1024,
		0,
		1024,
		0,
	)
	assertSingleContainsAll(t, message.Text, "速度：1.00 KB/s", "大小：1.00 KB / 未知")
	if strings.Contains(message.Text, "🟩") {
		t.Fatalf("unknown-size message should not show a misleading progress bar:\n%s", message.Text)
	}
}

func TestSingleDoneMessagesAndReplyMarkup(t *testing.T) {
	useSingleProgressTestLocale(t, "zh-Hans")
	info := progressTestTaskInfo{}
	success := buildSingleDoneMessage(info, info.FileSize(), nil)
	assertSingleContainsAll(t, success.Text,
		"✅ 处理完成",
		"文件名：file.bin",
		"总大小：100.00 MB",
		"保存至：[test]:file.bin",
	)
	canceled := buildSingleDoneMessage(info, info.FileSize(), context.Canceled)
	assertSingleContainsAll(t, canceled.Text, "🚫 任务已取消", "文件名：file.bin")
	failed := buildSingleDoneMessage(info, info.FileSize(), errors.New("remote timeout"))
	assertSingleContainsAll(t, failed.Text, "❌ 处理失败", "文件名：file.bin", "原因：remote timeout")

	finalRequest := buildSingleEditMessageRequest(1, "task", success, false)
	if finalRequest.ReplyMarkup != nil {
		t.Fatalf("final reply markup = %#v, want nil", finalRequest.ReplyMarkup)
	}
	activeRequest := buildSingleEditMessageRequest(1, "task", renderedSingleMessage{Text: "active"}, true)
	activeMarkup, ok := activeRequest.ReplyMarkup.(*tg.ReplyInlineMarkup)
	if !ok || len(activeMarkup.Rows) != 1 || len(activeMarkup.Rows[0].Buttons) != 1 {
		t.Fatalf("active reply markup = %#v, want one cancel button", activeRequest.ReplyMarkup)
	}
}

func TestSingleDoneMessageUsesActualUploadSize(t *testing.T) {
	useSingleProgressTestLocale(t, "zh-Hans")
	progress := new(Progress)
	info := progressTestTaskInfo{}
	progress.OnStart(context.Background(), info)
	progress.OnUploadStart(context.Background(), info, 2048)
	if got := progress.doneSize(info); got != 2048 {
		t.Fatalf("done size = %d, want actual upload size 2048", got)
	}
	message := buildSingleDoneMessage(info, progress.doneSize(info), nil)
	assertSingleContainsAll(t, message.Text, "总大小：2.00 KB")
}

func TestSingleProgressTextIsLocalized(t *testing.T) {
	useSingleProgressTestLocale(t, "en")
	message := buildSingleProgressMessage(progressTestTaskInfo{}, singlePhaseUploading, 50, 100, 25, 1)
	assertSingleContainsAll(t, message.Text, "📦 Processing", "⬆️ Uploading", "Speed: 25 B/s", "Size: 50 B / 100 B")
}

func assertSingleRichEntities(t *testing.T, message renderedSingleMessage, wantBlockquotes int) {
	t.Helper()
	var blockquotes, bold, code int
	for _, messageEntity := range message.Entities {
		switch messageEntity.(type) {
		case *tg.MessageEntityBlockquote:
			blockquotes++
		case *tg.MessageEntityBold:
			bold++
		case *tg.MessageEntityCode:
			code++
		}
	}
	if blockquotes != wantBlockquotes || bold < 2 || code == 0 {
		t.Fatalf("entity counts = blockquotes:%d bold:%d code:%d", blockquotes, bold, code)
	}
}

func useSingleProgressTestLocale(t *testing.T, language string) {
	t.Helper()
	i18n.Init(language)
	t.Cleanup(func() { i18n.Init("zh-Hans") })
}

func assertSingleContainsAll(t *testing.T, value string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(value, want) {
			t.Fatalf("text does not contain %q:\n%s", want, value)
		}
	}
}
