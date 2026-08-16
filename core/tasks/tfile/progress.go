package tfile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/common/utils/progressutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

type ProgressTracker interface {
	OnStart(ctx context.Context, info TaskInfo)
	OnProgress(ctx context.Context, info TaskInfo, downloaded, total int64)
	OnDone(ctx context.Context, info TaskInfo, err error)
}

// UploadProgressTracker optionally extends a task progress tracker with a
// distinct upload phase. Keeping it separate preserves compatibility with
// custom download-only trackers.
type UploadProgressTracker interface {
	OnUploadStart(ctx context.Context, info TaskInfo, total int64)
	OnUploadProgress(ctx context.Context, info TaskInfo, uploaded, total int64)
}

type Progress struct {
	MessageID         int
	ChatID            int64
	start             time.Time
	lastUpdatePercent atomic.Int32
	lastUpdateAt      atomic.Int64
	updateMu          sync.Mutex
	uploadAttempt     int
	uploadedBytes     int64
	actualSize        int64
	hasActualSize     bool
}

const (
	uploadProgressMinInterval = time.Second
	uploadProgressMaxInterval = 3 * time.Second
	singleProgressBarWidth    = 10
	maxSingleErrorRunes       = 240
)

type singleProgressPhase int

const (
	singlePhaseDownloading singleProgressPhase = iota
	singlePhaseUploading
	singlePhaseRetrying
)

type renderedSingleMessage struct {
	Text     string
	Entities []tg.MessageEntityClass
	Err      error
}

func (p *Progress) OnStart(ctx context.Context, info TaskInfo) {
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	p.start = time.Now()
	p.lastUpdatePercent.Store(0)
	p.lastUpdateAt.Store(0)
	p.uploadAttempt = 0
	p.uploadedBytes = 0
	p.actualSize = 0
	p.hasActualSize = false
	log.FromContext(ctx).Debugf("Progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)
	p.editMessage(ctx, info.TaskID(), buildSingleProgressMessage(info, singlePhaseDownloading, 0, info.FileSize(), 0, 0), true)
}

func (p *Progress) OnProgress(ctx context.Context, info TaskInfo, downloaded, total int64) {
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	now := time.Now()
	elapsed := uploadProgressMaxInterval
	if lastUpdateAt := p.lastUpdateAt.Load(); lastUpdateAt > 0 {
		elapsed = now.Sub(time.Unix(0, lastUpdateAt))
	}
	if !shouldUpdateSingleDownloadProgress(total, downloaded, int(p.lastUpdatePercent.Load()), elapsed) {
		return
	}
	if total > 0 {
		percent := int32((downloaded * 100) / total)
		if p.lastUpdatePercent.Load() == percent {
			return
		}
		p.lastUpdatePercent.Store(percent)
	}
	p.lastUpdateAt.Store(now.UnixNano())
	log.FromContext(ctx).Debugf("Progress update: %s, %d/%d", info.FileName(), downloaded, total)
	p.editMessage(ctx, info.TaskID(), buildSingleProgressMessage(
		info,
		singlePhaseDownloading,
		downloaded,
		total,
		dlutil.GetSpeed(downloaded, p.start),
		0,
	), true)
}

func shouldUpdateSingleDownloadProgress(total, downloaded int64, lastPercent int, elapsed time.Duration) bool {
	if total > 0 {
		return progressutil.ShouldUpdate(total, downloaded, lastPercent)
	}
	return downloaded > 0 && elapsed >= uploadProgressMaxInterval
}

func (p *Progress) OnUploadStart(ctx context.Context, info TaskInfo, total int64) {
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	p.start = time.Now()
	p.lastUpdatePercent.Store(0)
	p.lastUpdateAt.Store(p.start.UnixNano())
	p.uploadAttempt++
	p.uploadedBytes = 0
	p.actualSize = max(total, 0)
	p.hasActualSize = true
	log.FromContext(ctx).Debugf("Upload progress tracking started: %s", info.FileName())
	phase := singleUploadPhase(p.uploadAttempt)
	p.editMessage(ctx, info.TaskID(), buildSingleProgressMessage(info, phase, 0, total, 0, p.uploadAttempt), true)
}

func (p *Progress) OnUploadProgress(ctx context.Context, info TaskInfo, uploaded, total int64) {
	if total <= 0 || uploaded <= 0 {
		return
	}
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	if uploaded > total {
		uploaded = total
	}
	if uploaded < p.uploadedBytes {
		return
	}
	p.uploadedBytes = uploaded

	now := time.Now()
	lastUpdateAt := time.Unix(0, p.lastUpdateAt.Load())
	lastPercent := int(p.lastUpdatePercent.Load())
	if !shouldUpdateUploadProgress(total, uploaded, lastPercent, now.Sub(lastUpdateAt)) {
		return
	}

	percent := int32((uploaded * 100) / total)
	p.lastUpdatePercent.Store(percent)
	p.lastUpdateAt.Store(now.UnixNano())
	log.FromContext(ctx).Debugf("Upload progress update: %s, %d/%d", info.FileName(), uploaded, total)
	p.editMessage(ctx, info.TaskID(), buildSingleProgressMessage(
		info,
		singleUploadPhase(p.uploadAttempt),
		uploaded,
		total,
		dlutil.GetSpeed(uploaded, p.start),
		p.uploadAttempt,
	), true)
}

func shouldUpdateUploadProgress(total, uploaded int64, lastPercent int, elapsed time.Duration) bool {
	if total <= 0 || uploaded <= 0 {
		return false
	}
	if uploaded >= total {
		return lastPercent < 100 && elapsed >= uploadProgressMinInterval
	}
	percent := int((uploaded * 100) / total)
	if percent < lastPercent {
		return false
	}
	if elapsed < uploadProgressMinInterval {
		return false
	}
	if percent == lastPercent {
		return elapsed >= uploadProgressMaxInterval
	}
	return progressutil.ShouldUpdate(total, uploaded, lastPercent) || elapsed >= uploadProgressMaxInterval
}

func singleUploadPhase(attempt int) singleProgressPhase {
	if attempt > 1 {
		return singlePhaseRetrying
	}
	return singlePhaseUploading
}

func (p *Progress) OnDone(ctx context.Context, info TaskInfo, err error) {
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	if err != nil {
		log.FromContext(ctx).Errorf("Progress error for file [%s]: %v", info.FileName(), err)
	} else {
		log.FromContext(ctx).Debugf("Progress done for file [%s]", info.FileName())
	}

	p.editMessage(ctx, info.TaskID(), buildSingleDoneMessage(info, p.doneSize(info), err), false)
}

func (p *Progress) doneSize(info TaskInfo) int64 {
	if p.hasActualSize {
		return p.actualSize
	}
	return max(info.FileSize(), 0)
}

func (p *Progress) editMessage(ctx context.Context, taskID string, message renderedSingleMessage, cancellable bool) {
	if message.Err != nil {
		log.FromContext(ctx).Errorf("Failed to render file progress message: %v", message.Err)
		return
	}
	req := buildSingleEditMessageRequest(p.MessageID, taskID, message, cancellable)
	if ext := tgutil.ExtFromContext(ctx); ext != nil {
		if _, err := ext.EditMessage(p.ChatID, req); err != nil {
			log.FromContext(ctx).Errorf("Failed to edit file progress message: %v", err)
		}
	}
}

func buildSingleEditMessageRequest(messageID int, taskID string, message renderedSingleMessage, cancellable bool) *tg.MessagesEditMessageRequest {
	req := &tg.MessagesEditMessageRequest{ID: messageID}
	req.SetMessage(message.Text)
	if len(message.Entities) > 0 {
		req.SetEntities(message.Entities)
	}
	if cancellable {
		req.SetReplyMarkup(&tg.ReplyInlineMarkup{Rows: []tg.KeyboardButtonRow{{
			Buttons: []tg.KeyboardButtonClass{tgutil.BuildCancelButton(taskID)},
		}}})
	}
	return req
}

func buildSingleProgressMessage(
	info TaskInfo,
	phase singleProgressPhase,
	current int64,
	total int64,
	speed float64,
	attempt int,
) renderedSingleMessage {
	if current < 0 {
		current = 0
	}
	if total > 0 && current > total {
		current = total
	}
	percent := singleProgressPercent(current, total)
	destination := fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())
	data := map[string]any{
		"Name":        info.FileName(),
		"Bar":         singleProgressBar(percent),
		"Progress":    percent,
		"Speed":       singleProgressSpeed(speed),
		"Current":     dlutil.FormatSize(current),
		"Size":        dlutil.FormatSize(total),
		"Destination": destination,
		"Attempt":     max(attempt, 1),
	}

	var key i18nk.Key
	switch phase {
	case singlePhaseUploading:
		key = i18nk.BotMsgProgressSingleUploading
	case singlePhaseRetrying:
		key = i18nk.BotMsgProgressSingleUploadRetrying
	default:
		key = i18nk.BotMsgProgressSingleDownloading
		if total <= 0 {
			key = i18nk.BotMsgProgressSingleDownloadingUnknown
		}
	}
	markup := i18n.T(i18nk.BotMsgProgressSingleStatusHeader, nil) + "\n\n" + localizedProgressMarkup(key, data)
	return completeSingleMessage(markup)
}

func buildSingleDoneMessage(info TaskInfo, size int64, err error) renderedSingleMessage {
	data := map[string]any{
		"Name":        info.FileName(),
		"Size":        dlutil.FormatSize(max(size, 0)),
		"Destination": fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath()),
	}
	var key i18nk.Key
	switch {
	case err == nil:
		key = i18nk.BotMsgProgressSingleDone
	case errors.Is(err, context.Canceled):
		key = i18nk.BotMsgProgressSingleCanceled
	default:
		data["Reason"] = truncateSingleError(err.Error())
		key = i18nk.BotMsgProgressSingleFailed
	}
	return completeSingleMessage(localizedProgressMarkup(key, data))
}

func localizedProgressMarkup(key i18nk.Key, data map[string]any) string {
	return i18n.T(key, tgutil.EscapeHTMLTemplateData(data))
}

func completeSingleMessage(markup string) renderedSingleMessage {
	text, entities, err := tgutil.RenderHTML(markup)
	return renderedSingleMessage{Text: text, Entities: entities, Err: err}
}

func singleProgressPercent(current, total int64) int {
	if total <= 0 {
		return 0
	}
	return int(min(max(current, 0), total) * 100 / total)
}

func singleProgressBar(percent int) string {
	percent = min(max(percent, 0), 100)
	filled := percent * singleProgressBarWidth / 100
	return strings.Repeat("🟩", filled) + strings.Repeat("⬜️", singleProgressBarWidth-filled)
}

func singleProgressSpeed(speed float64) string {
	if speed <= 0 {
		return "0 B/s"
	}
	return dlutil.FormatSize(int64(speed)) + "/s"
}

func truncateSingleError(value string) string {
	runes := []rune(value)
	if len(runes) <= maxSingleErrorRunes {
		return value
	}
	return string(runes[:maxSingleErrorRunes])
}

type ProgressOption func(*Progress)

func NewProgressTrack(
	messageID int,
	chatID int64,
	opts ...ProgressOption,
) ProgressTracker {
	p := &Progress{
		MessageID: messageID,
		ChatID:    chatID,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}
