package tfile

import (
	"context"
	"errors"
	"fmt"
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
		wg.Go(func() {
			progress.OnUploadProgress(ctx, info, uploaded, total)
		})
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

func TestSingleUploadRetryKeepsRichProgressLayout(t *testing.T) {
	i18n.Init("zh-Hans")
	t.Cleanup(func() { i18n.Init("zh-Hans") })

	message := buildSingleProgressMessage(
		progressTestTaskInfo{},
		singleUploadPhase(2),
		25<<20,
		100<<20,
		5<<20,
		2,
	)
	if message.Err != nil {
		t.Fatalf("buildSingleProgressMessage() failed: %v", message.Err)
	}
	for _, want := range []string{
		"🔁 上传重试",
		"🟩🟩⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️ 25%",
		"尝试次数：2",
		"速度：5.00 MB/s",
		"大小：25.00 MB / 100.00 MB",
	} {
		if !strings.Contains(message.Text, want) {
			t.Fatalf("retry progress does not contain %q:\n%s", want, message.Text)
		}
	}
}

func TestSingleProgressTemplateOwnsStylesAndEscapesValues(t *testing.T) {
	i18n.Init("en")
	t.Cleanup(func() { i18n.Init("zh-Hans") })
	info := htmlProgressTestTaskInfo{}

	message := buildSingleProgressMessage(info, singlePhaseDownloading, 50, 100, 25, 0)
	if message.Err != nil {
		t.Fatalf("buildSingleProgressMessage() failed: %v", message.Err)
	}
	for _, want := range []string{
		`<b>A&B</b>.bin`,
		`[store<&>]:dir/<i>x</i>&`,
		"Speed: 25 B/s",
	} {
		if !strings.Contains(message.Text, want) {
			t.Fatalf("progress text does not contain %q:\n%s", want, message.Text)
		}
	}
	bold, code, blockquote, italic := singleEntityCounts(message.Entities)
	if bold != 2 || code != 6 || blockquote != 1 || italic != 0 {
		t.Fatalf("progress entity counts = bold:%d code:%d blockquote:%d italic:%d", bold, code, blockquote, italic)
	}

	failure := buildSingleDoneMessage(context.Background(), info, 100, errors.New(`<i>remote & failed</i>`))
	if failure.Err != nil {
		t.Fatalf("buildSingleDoneMessage() failed: %v", failure.Err)
	}
	if !strings.Contains(failure.Text, `<i>remote & failed</i>`) {
		t.Fatalf("failure reason was not preserved literally:\n%s", failure.Text)
	}
	bold, code, blockquote, italic = singleEntityCounts(failure.Entities)
	if bold != 1 || code != 2 || blockquote != 0 || italic != 0 {
		t.Fatalf("failure entity counts = bold:%d code:%d blockquote:%d italic:%d", bold, code, blockquote, italic)
	}
}

func TestSingleDoneMessageUsesTaskContext(t *testing.T) {
	i18n.Init("zh-Hans")
	t.Cleanup(func() { i18n.Init("zh-Hans") })
	canceledCtx, cancel := context.WithCancel(t.Context())
	cancel()
	timedOutCtx, timeoutCancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer timeoutCancel()
	engineErr := fmt.Errorf("engine forcibly closed: %w", context.Canceled)
	for _, tt := range []struct {
		name         string
		ctx          context.Context
		err          error
		want         string
		codeEntities int
	}{
		{"engine closed with active task", t.Context(), engineErr, "处理失败", 2},
		{"task canceled", canceledCtx, engineErr, "任务已取消", 1},
		{"task canceled with other error", canceledCtx, errors.New("connection EOF"), "任务已取消", 1},
		{"task timed out", timedOutCtx, context.DeadlineExceeded, "处理失败", 2},
		{"ordinary failure", t.Context(), errors.New("connection EOF"), "处理失败", 2},
		{"success", t.Context(), nil, "处理完成", 3},
		{"success before cancellation", canceledCtx, nil, "处理完成", 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			message := buildSingleDoneMessage(tt.ctx, progressTestTaskInfo{}, 100, tt.err)
			if message.Err != nil {
				t.Fatal(message.Err)
			}
			if !strings.Contains(message.Text, tt.want) {
				t.Fatalf("message = %q, want %q", message.Text, tt.want)
			}
			if tt.want == "处理失败" && !strings.Contains(message.Text, tt.err.Error()) {
				t.Fatalf("failure reason missing from %q", message.Text)
			}
			bold, code, blockquote, italic := singleEntityCounts(message.Entities)
			if bold != 1 || code != tt.codeEntities || blockquote != 0 || italic != 0 {
				t.Fatalf("entity counts = %d/%d/%d/%d, want 1/%d/0/0", bold, code, blockquote, italic, tt.codeEntities)
			}
		})
	}
}

func TestSingleDoneSizeUsesActualUploadSize(t *testing.T) {
	progress := new(Progress)
	info := progressTestTaskInfo{}
	progress.OnStart(context.Background(), info)
	progress.OnUploadStart(context.Background(), info, 2048)

	if got := progress.doneSize(info); got != 2048 {
		t.Fatalf("done size = %d, want actual upload size 2048", got)
	}
}

type htmlProgressTestTaskInfo struct{}

func (htmlProgressTestTaskInfo) TaskID() string      { return "html-task" }
func (htmlProgressTestTaskInfo) FileName() string    { return `<b>A&B</b>.bin` }
func (htmlProgressTestTaskInfo) FileSize() int64     { return 100 }
func (htmlProgressTestTaskInfo) StoragePath() string { return `dir/<i>x</i>&` }
func (htmlProgressTestTaskInfo) StorageName() string { return `store<&>` }

func singleEntityCounts(entities []tg.MessageEntityClass) (bold, code, blockquote, italic int) {
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
