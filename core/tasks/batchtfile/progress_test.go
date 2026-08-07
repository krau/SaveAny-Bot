package batchtfile

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
)

func TestBatchProgressShowsConcurrentDownloadAndUploadSpeeds(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(
		progressTestFile{"completed", 1000},
		progressTestFile{"downloading", 1000},
		progressTestFile{"uploading", 1000},
		progressTestFile{"waiting", 1000},
	)
	started := time.Unix(100, 0)

	task.markItemCompleted("completed")
	task.markItemActive("downloading", false, started)
	task.recordItemDownload("downloading", 500, started.Add(time.Second))
	task.markItemActive("uploading", false, started)
	task.recordItemDownload("uploading", 1000, started.Add(time.Second))
	task.recordItemDownloaded("uploading", 1000)
	task.recordItemUpload("uploading", 0, 1000, started.Add(time.Second))
	task.recordItemUpload("uploading", 250, 1000, started.Add(2*time.Second))

	got := buildBatchProgressText(task, nil, 3)
	assertContainsAll(t, got,
		"文件：4",
		"状态：✅ 1 ｜ 📥 0 ｜ ⏳ 1",
		"总速度：⬇️ 500 B/s ｜ ⬆️ 250 B/s",
		"⬇️ 2/4 下载中",
		"downloading.bin",
		"🟩🟩🟩🟩🟩⬜️⬜️⬜️⬜️⬜️ 50%",
		"⬆️ 3/4 上传中",
		"uploading.bin",
		"🟩🟩⬜️⬜️⬜️⬜️⬜️⬜️⬜️⬜️ 25%",
		"速度：500 B/s",
		"大小：500 B / 1000 B",
		"速度：250 B/s",
		"大小：250 B / 1000 B",
	)
}

func TestBatchProgressLimitsActiveRowsAndCompressesTheRest(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(
		progressTestFile{"file-01", 100},
		progressTestFile{"file-02", 100},
		progressTestFile{"file-03", 100},
		progressTestFile{"file-04", 100},
		progressTestFile{"file-05", 100},
		progressTestFile{"file-06", 100},
		progressTestFile{"file-07", 100},
	)
	for _, item := range task.Items() {
		task.markItemActive(item.ID, false, time.Unix(100, 0))
	}

	got := buildBatchProgressText(task, nil, 3)
	assertContainsAll(t, got, "file-01.bin", "file-02.bin", "file-03.bin", "🔄 另有 4 个文件正在处理")
	if strings.Contains(got, "file-04.bin") {
		t.Fatalf("fourth active item should be compressed:\n%s", got)
	}
}

func TestBatchProgressCompressesDownloadedItemsWaitingForBatchUpload(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(
		progressTestFile{"downloaded-01", 100},
		progressTestFile{"downloaded-02", 100},
		progressTestFile{"active", 100},
		progressTestFile{"waiting", 100},
	)
	task.markItemDownloaded("downloaded-01")
	task.markItemDownloaded("downloaded-02")
	task.markItemActive("active", false, time.Unix(100, 0))

	got := buildBatchProgressText(task, nil, 3)
	assertContainsAll(t, got,
		"状态：✅ 0 ｜ 📥 2 ｜ ⏳ 1",
		"active.bin",
	)
	if strings.Contains(got, "downloaded-01.bin") || strings.Contains(got, "downloaded-02.bin") {
		t.Fatalf("downloaded items should be compressed:\n%s", got)
	}
}

func TestBatchProgressShowsRetryAndConfirmationStates(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(progressTestFile{"retry", 100}, progressTestFile{"confirm", 100})
	task.markItemRetry("retry", FailureStageUpload, 1, 3, errors.New("  remote   timeout\n"))
	task.recordItemUpload("confirm", 100, 100, time.Unix(100, 0))

	got := buildBatchProgressText(task, nil, 2)
	assertContainsAll(t, got,
		"上传重试",
		"重试次数：1/3",
		"大小：0 B / 100 B",
		"原因：remote timeout",
		"⏳ 2/2 等待中",
		"文件已上传，正在等待远端确认",
	)
}

func TestBatchProgressUnknownSizeShowsTransferredBytes(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(progressTestFile{"unknown", 0})
	started := time.Unix(100, 0)
	task.markItemActive("unknown", false, started)
	task.recordItemDownload("unknown", 1024, started.Add(time.Second))

	got := buildBatchProgressText(task, nil, 1)
	assertContainsAll(t, got, "速度：1.00 KB/s", "大小：1.00 KB / 未知")
	if strings.Index(got, "速度：") > strings.Index(got, "大小：") {
		t.Fatalf("size should be shown below speed:\n%s", got)
	}
}

func TestBatchProgressTransferIsNotHiddenByConfirmingItems(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(
		progressTestFile{"confirm-01", 100},
		progressTestFile{"confirm-02", 100},
		progressTestFile{"uploading", 100},
	)
	started := time.Unix(100, 0)
	task.recordItemUpload("confirm-01", 100, 100, started)
	task.recordItemUpload("confirm-02", 100, 100, started)
	task.recordItemUpload("uploading", 40, 100, started.Add(time.Second))

	message := buildBatchProgressMessage(task, nil, 2)
	assertContainsAll(t, message.Text,
		"uploading.bin",
		"🟩🟩🟩🟩⬜️⬜️⬜️⬜️⬜️⬜️ 40%",
		"☁️ 已上传，等待整组发送：2",
	)
	if strings.Contains(message.Text, "confirm-01.bin") || strings.Contains(message.Text, "confirm-02.bin") {
		t.Fatalf("confirming items hid the active upload:\n%s", message.Text)
	}
	blockquoteCount := 0
	for _, messageEntity := range message.Entities {
		if _, ok := messageEntity.(*tg.MessageEntityBlockquote); ok {
			blockquoteCount++
		}
	}
	if blockquoteCount != 1 {
		t.Fatalf("blockquote count = %d, want only the active upload", blockquoteCount)
	}
}

func TestBatchProgressUsesSeparateBlockquotesForActiveItems(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(progressTestFile{"first", 100}, progressTestFile{"second", 100})
	for _, item := range task.Items() {
		task.markItemActive(item.ID, false, time.Unix(100, 0))
	}

	message := buildBatchProgressMessage(task, nil, 2)
	blockquoteCount := 0
	boldCount := 0
	codeCount := 0
	for _, messageEntity := range message.Entities {
		switch messageEntity.(type) {
		case *tg.MessageEntityBlockquote:
			blockquoteCount++
		case *tg.MessageEntityBold:
			boldCount++
		case *tg.MessageEntityCode:
			codeCount++
		}
	}
	if blockquoteCount != 2 {
		t.Fatalf("blockquote count = %d, want 2", blockquoteCount)
	}
	if boldCount < 3 {
		t.Fatalf("bold entity count = %d, want header plus two item headings", boldCount)
	}
	if codeCount == 0 {
		t.Fatal("expected code entities for filenames and progress values")
	}
}

func TestTextProgressBarUsesTelegramEmojiCells(t *testing.T) {
	if got, want := textProgressBar(50), "🟩🟩🟩🟩🟩⬜️⬜️⬜️⬜️⬜️"; got != want {
		t.Fatalf("textProgressBar(50) = %q, want %q", got, want)
	}
	if got, want := textProgressBar(100), "🟩🟩🟩🟩🟩🟩🟩🟩🟩🟩"; got != want {
		t.Fatalf("textProgressBar(100) = %q, want %q", got, want)
	}
}

func TestConfirmationStateBypassesProgressThrottle(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	progress := new(Progress)
	task := progressTestTask(progressTestFile{"file", 100})
	task.Progress = progress
	task.recordDownloadComplete("file", 100)
	progress.OnStart(t.Context(), task)

	task.uploadCallback(t.Context(), "file")(100, 100)
	progress.updateMu.Lock()
	text := progress.lastText
	progress.updateMu.Unlock()
	if !strings.Contains(text, "文件已上传，正在等待远端确认") {
		t.Fatalf("confirmation state was rate limited:\n%s", text)
	}
}

func TestBatchFinalEditOmitsInvalidEmptyReplyMarkup(t *testing.T) {
	message := renderedBatchMessage{
		Text:     "done",
		Entities: []tg.MessageEntityClass{&tg.MessageEntityBold{Offset: 0, Length: 4}},
	}
	finalRequest := buildBatchEditMessageRequest(1, "task", message, false)
	if finalRequest.ReplyMarkup != nil {
		t.Fatalf("final reply markup = %#v, want nil", finalRequest.ReplyMarkup)
	}
	if len(finalRequest.Entities) != 1 {
		t.Fatalf("final entities = %#v, want one bold entity", finalRequest.Entities)
	}

	activeRequest := buildBatchEditMessageRequest(1, "task", renderedBatchMessage{Text: "active"}, true)
	activeMarkup, ok := activeRequest.ReplyMarkup.(*tg.ReplyInlineMarkup)
	if !ok || len(activeMarkup.Rows) != 1 || len(activeMarkup.Rows[0].Buttons) != 1 {
		t.Fatalf("active reply markup = %#v, want one cancel button", activeRequest.ReplyMarkup)
	}
}

func TestBatchDoneMessagesCoverSuccessSkippedAndCancel(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(progressTestFile{"unknown", 0}, progressTestFile{"known", 2048})
	task.markItemActive("unknown", false, time.Unix(100, 0))
	task.recordItemDownload("unknown", 1024, time.Unix(101, 0))
	task.markItemCompleted("unknown")
	task.markItemCompleted("known")

	success := buildBatchDoneText(task, nil, nil)
	assertContainsAll(t, success, "✅ 处理完成", "文件数: 2", "总大小: 3.00 KB")
	successMessage := buildBatchDoneMessage(task, nil, nil)
	if successMessage.Text != success {
		t.Fatalf("rendered success text = %q, want %q", successMessage.Text, success)
	}
	var bold, code int
	for _, messageEntity := range successMessage.Entities {
		switch messageEntity.(type) {
		case *tg.MessageEntityBold:
			bold++
		case *tg.MessageEntityCode:
			code++
		}
	}
	if bold != 1 || code != 2 {
		t.Fatalf("success entity counts = bold:%d code:%d, want bold:1 code:2", bold, code)
	}

	withSkipped := buildBatchDoneText(task, []string{"skipped.bin"}, nil)
	assertContainsAll(t, withSkipped, "⚠️ 处理完成", "成功: 2", "已跳过: 1", "总大小: 3.00 KB")

	canceled := progressTestTask(progressTestFile{"done", 100}, progressTestFile{"waiting", 100})
	canceled.markItemCompleted("done")
	canceled.finishItems(context.Canceled)
	canceledText := buildBatchDoneText(canceled, []string{"skipped.bin"}, context.Canceled)
	assertContainsAll(t, canceledText, "🚫 任务已取消", "文件数: 3", "已完成: 1", "未完成: 1", "已跳过: 1")
}

func TestBatchDoneMessageShowsPerFileFailureDetails(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(
		progressTestFile{"done", 100},
		progressTestFile{"broken", 100},
		progressTestFile{"waiting", 100},
	)
	task.markItemCompleted("done")
	started := time.Unix(100, 0)
	task.markItemActive("broken", false, started)
	task.recordItemDownload("broken", 40, started.Add(time.Second))
	task.markItemFailed("broken", FailureStageDownload, errors.New("telegram   connection lost"))

	got := buildBatchDoneText(task, nil, errors.New("wrapped task error"))
	assertContainsAll(t, got,
		"❌ 处理失败",
		"失败文件: 2. broken.bin",
		"失败阶段: 下载",
		"失败进度: 40%",
		"失败前速度: 40 B/s",
		"原因: telegram connection lost",
		"✅ 已完成: 1",
		"❌ 失败: 1",
		"⏹️ 未完成: 1",
	)
}

func TestBatchDoneMessageShowsGroupFailure(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(progressTestFile{"first", 100}, progressTestFile{"second", 100})
	err := errors.New("album rejected")
	task.markItemFailed("first", FailureStageBatchUpload, err)
	task.markItemFailed("second", FailureStageBatchUpload, err)

	got := buildBatchDoneText(task, nil, err)
	assertContainsAll(t, got,
		"❌ 批量上传失败",
		"受影响文件: 2 个",
		"原因: album rejected",
		"❌ 批次失败: 2",
	)
}

func TestBatchDoneMessagePreservesUploadSpeedBeforeFailure(t *testing.T) {
	useProgressTestLocale(t, "zh-Hans")
	task := progressTestTask(progressTestFile{"broken", 1000})
	started := time.Unix(100, 0)
	task.markItemActive("broken", false, started)
	task.recordItemDownload("broken", 1000, started.Add(time.Second))
	task.recordItemDownloaded("broken", 1000)
	task.recordItemUpload("broken", 0, 1000, started.Add(time.Second))
	task.recordItemUpload("broken", 500, 1000, started.Add(2*time.Second))
	err := errors.New("remote timeout")
	task.markItemRetry("broken", FailureStageUpload, 3, 3, err)
	task.markItemFailed("broken", FailureStageUpload, err)

	got := buildBatchDoneText(task, nil, err)
	assertContainsAll(t, got, "失败阶段: 上传", "失败进度: 50%", "失败前速度: 500 B/s")
}

func TestBatchProgressTextIsLocalized(t *testing.T) {
	useProgressTestLocale(t, "en")
	task := progressTestTask(progressTestFile{"file", 100})

	got := buildBatchProgressText(task, nil, 1)
	assertContainsAll(t, got, "📦 Processing", "Files: 1", "Status: ✅ 0 | 📥 0 | ⏳ 1")
}

func useProgressTestLocale(t *testing.T, language string) {
	t.Helper()
	i18n.Init(language)
	t.Cleanup(func() { i18n.Init("zh-Hans") })
}

func assertContainsAll(t *testing.T, value string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(value, want) {
			t.Fatalf("text does not contain %q:\n%s", want, value)
		}
	}
}
