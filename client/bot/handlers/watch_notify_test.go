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

func TestWatchNotifyProgressReportsThroughBotContext(t *testing.T) {
	userbotExt := &ext.Context{}
	botExt := &ext.Context{}
	inner := &recordingTracker{}
	tracker := watchNotifyProgress{ext: botExt, tracker: inner}

	taskCtx := context.WithValue(context.Background(), notifyCtxKey{}, "kept")
	taskCtx = tgutil.ExtWithContext(taskCtx, userbotExt)

	tracker.OnStart(taskCtx, nil)
	tracker.OnProgress(taskCtx, nil, 1, 2)
	tracker.OnUploadStart(taskCtx, nil, 3)
	tracker.OnUploadProgress(taskCtx, nil, 1, 3)
	tracker.OnDone(taskCtx, nil, errors.New("failed"))

	wantEvents := []string{"start", "progress", "upload_start", "upload_progress", "done"}
	if !slices.Equal(inner.events, wantEvents) {
		t.Fatalf("forwarded events = %q, want %q", inner.events, wantEvents)
	}
	for i, ctx := range inner.ctxs {
		if got := tgutil.ExtFromContext(ctx); got != botExt {
			t.Errorf("event %s used ext %p, want the bot ext %p", inner.events[i], got, botExt)
		}
		if got := ctx.Value(notifyCtxKey{}); got != "kept" {
			t.Errorf("event %s dropped task context values: %v", inner.events[i], got)
		}
	}
	if got := tgutil.ExtFromContext(taskCtx); got != userbotExt {
		t.Errorf("task context ext = %p, want the userbot ext %p", got, userbotExt)
	}
}

func TestNewWatchNotifyProgressDisabled(t *testing.T) {
	botExt := &ext.Context{}
	if tracker := newWatchNotifyProgress(t.Context(), botExt, &database.User{}, nil); tracker != nil {
		t.Error("tracker created while notifications are disabled")
	}
	if tracker := newWatchNotifyProgress(t.Context(), nil, &database.User{WatchNotify: true}, nil); tracker != nil {
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
