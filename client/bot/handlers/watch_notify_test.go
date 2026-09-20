package handlers

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"

	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	tftask "github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
)

func TestParseWatchNotifyArg(t *testing.T) {
	tests := []struct {
		arg         string
		wantEnabled bool
		wantOK      bool
	}{
		{"on", true, true},
		{"ON", true, true},
		{"off", false, true},
		{"Off", false, true},
		{"", false, false},
		{"yes", false, false},
		{"-1002229835658", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			enabled, ok := parseWatchNotifyArg(tt.arg)
			if enabled != tt.wantEnabled || ok != tt.wantOK {
				t.Errorf("parseWatchNotifyArg(%q) = (%v, %v), want (%v, %v)", tt.arg, enabled, ok, tt.wantEnabled, tt.wantOK)
			}
		})
	}
}

type notifyCtxKey struct{}

type notifyTestInfo struct{}

func (notifyTestInfo) TaskID() string      { return "notify-task" }
func (notifyTestInfo) FileName() string    { return "file.bin" }
func (notifyTestInfo) FileSize() int64     { return 1 }
func (notifyTestInfo) StoragePath() string { return "dir/file.bin" }
func (notifyTestInfo) StorageName() string { return "store" }

// recordingTracker records the contexts the wrapped tracker is called with.
type recordingTracker struct {
	events []string
	ctxs   []context.Context
}

func (r *recordingTracker) OnStart(ctx context.Context, _ tftask.TaskInfo) {
	r.events = append(r.events, "start")
	r.ctxs = append(r.ctxs, ctx)
}

func (r *recordingTracker) OnProgress(ctx context.Context, _ tftask.TaskInfo, _, _ int64) {
	r.events = append(r.events, "progress")
	r.ctxs = append(r.ctxs, ctx)
}

func (r *recordingTracker) OnUploadStart(ctx context.Context, _ tftask.TaskInfo, _ int64) {
	r.events = append(r.events, "upload_start")
	r.ctxs = append(r.ctxs, ctx)
}

func (r *recordingTracker) OnUploadProgress(ctx context.Context, _ tftask.TaskInfo, _, _ int64) {
	r.events = append(r.events, "upload_progress")
	r.ctxs = append(r.ctxs, ctx)
}

func (r *recordingTracker) OnDone(ctx context.Context, _ tftask.TaskInfo, _ error) {
	r.events = append(r.events, "done")
	r.ctxs = append(r.ctxs, ctx)
}

func TestWatchNotifyProgressOpensNotificationOnFirstUse(t *testing.T) {
	userbotExt := &ext.Context{}
	sharedBotExt := &ext.Context{}
	inner := &recordingTracker{}

	var opened int
	var gotChatID int64
	var gotName string
	restore := openWatchNotification
	openWatchNotification = func(_ context.Context, _ *ext.Context, chatID int64, fileName string) tftask.ProgressTracker {
		opened++
		gotChatID = chatID
		gotName = fileName
		return inner
	}
	t.Cleanup(func() { openWatchNotification = restore })

	tracker := newWatchNotifyProgress(context.Background(), sharedBotExt, &database.User{WatchNotify: true, ChatID: 4242})
	progress, ok := tracker.(*watchNotifyProgress)
	if !ok {
		t.Fatalf("newWatchNotifyProgress() = %T, want *watchNotifyProgress", tracker)
	}
	if progress.botCtx == sharedBotExt {
		t.Fatal("notification reuses the shared bot context, which is not safe for concurrent sends")
	}
	other, ok := newWatchNotifyProgress(context.Background(), sharedBotExt, &database.User{WatchNotify: true, ChatID: 1}).(*watchNotifyProgress)
	if !ok {
		t.Fatal("second notification tracker has an unexpected type")
	}
	if other.botCtx == progress.botCtx {
		t.Error("notifications share one bot context")
	}

	taskCtx := context.WithValue(context.Background(), notifyCtxKey{}, "kept")
	taskCtx = tgutil.ExtWithContext(taskCtx, userbotExt)

	if opened != 0 {
		t.Fatalf("notification opened %d times before the task reported anything", opened)
	}

	uploads, ok := tracker.(tftask.UploadProgressTracker)
	if !ok {
		t.Fatal("tracker no longer reports upload progress")
	}
	tracker.OnStart(taskCtx, notifyTestInfo{})
	tracker.OnProgress(taskCtx, notifyTestInfo{}, 1, 2)
	uploads.OnUploadStart(taskCtx, notifyTestInfo{}, 3)
	uploads.OnUploadProgress(taskCtx, notifyTestInfo{}, 1, 3)
	tracker.OnDone(taskCtx, notifyTestInfo{}, errors.New("failed"))

	if opened != 1 {
		t.Errorf("notification opened %d times, want 1", opened)
	}
	if gotChatID != 4242 {
		t.Errorf("notification chat = %d, want 4242", gotChatID)
	}
	if want := (notifyTestInfo{}).FileName(); gotName != want {
		t.Errorf("notification file name = %q, want %q", gotName, want)
	}

	wantEvents := []string{"start", "progress", "upload_start", "upload_progress", "done"}
	if !slices.Equal(inner.events, wantEvents) {
		t.Fatalf("forwarded events = %q, want %q", inner.events, wantEvents)
	}
	for i, ctx := range inner.ctxs {
		if got := tgutil.ExtFromContext(ctx); got != progress.botCtx {
			t.Errorf("event %s used ext %p, want the per-task bot context %p", inner.events[i], got, progress.botCtx)
		}
		if got := ctx.Value(notifyCtxKey{}); got != "kept" {
			t.Errorf("event %s dropped task context values: %v", inner.events[i], got)
		}
	}
	if got := tgutil.ExtFromContext(taskCtx); got != userbotExt {
		t.Errorf("task context ext = %p, want the userbot ext %p", got, userbotExt)
	}
}

func TestWatchNotifyProgressStaysSilentWhenNotificationFails(t *testing.T) {
	restore := openWatchNotification
	opened := 0
	openWatchNotification = func(context.Context, *ext.Context, int64, string) tftask.ProgressTracker {
		opened++
		return nil
	}
	t.Cleanup(func() { openWatchNotification = restore })

	tracker := newWatchNotifyProgress(context.Background(), &ext.Context{}, &database.User{WatchNotify: true, ChatID: 1})
	tracker.OnStart(context.Background(), notifyTestInfo{})
	tracker.OnProgress(context.Background(), notifyTestInfo{}, 1, 1)
	tracker.OnDone(context.Background(), notifyTestInfo{}, nil)

	if opened != 1 {
		t.Errorf("notification opened %d times, want 1", opened)
	}
}

func TestNewWatchNotifyProgressDisabled(t *testing.T) {
	restore := openWatchNotification
	openWatchNotification = func(context.Context, *ext.Context, int64, string) tftask.ProgressTracker {
		t.Error("opened a notification while it is disabled")
		return nil
	}
	t.Cleanup(func() { openWatchNotification = restore })

	if tracker := newWatchNotifyProgress(t.Context(), &ext.Context{}, &database.User{}); tracker != nil {
		t.Error("tracker created while notifications are disabled")
	}
	if tracker := newWatchNotifyProgress(t.Context(), nil, &database.User{WatchNotify: true}); tracker != nil {
		t.Error("tracker created without a bot context")
	}
}

func TestWatchNotifyMarkupEscapesFileName(t *testing.T) {
	i18n.Init("en")
	t.Cleanup(func() { i18n.Init("zh-Hans") })

	text, entities, err := watchNotifyMarkup(`<b>A&B</b>.bin`)
	if err != nil {
		t.Fatalf("watchNotifyMarkup() failed: %v", err)
	}
	if !strings.Contains(text, `<b>A&B</b>.bin`) {
		t.Fatalf("file name was not preserved literally:\n%s", text)
	}
	var bold, code int
	for _, entity := range entities {
		switch entity.(type) {
		case *tg.MessageEntityBold:
			bold++
		case *tg.MessageEntityCode:
			code++
		}
	}
	if bold != 1 || code != 1 {
		t.Fatalf("entity counts = bold:%d code:%d, want bold:1 code:1", bold, code)
	}
}
