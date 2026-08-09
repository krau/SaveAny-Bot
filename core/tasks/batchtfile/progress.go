package batchtfile

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
)

type ProgressTracker interface {
	OnStart(ctx context.Context, info TaskInfo)
	OnProgress(ctx context.Context, info TaskInfo)
	OnDone(ctx context.Context, info TaskInfo, err error)
}

type Progress struct {
	MessageID    int
	ChatID       int64
	updateMu     sync.Mutex
	lastUpdateAt time.Time
	lastText     string
	done         bool
	skippedFiles []string
}

type renderedBatchMessage struct {
	Text     string
	Entities []tg.MessageEntityClass
	Err      error
}

const (
	progressRenderInterval = time.Second
	maxVisibleActiveItems  = 5
	progressBarWidth       = 10
	maxDisplayNameRunes    = 36
	maxDisplayErrorRunes   = 240
)

func (p *Progress) OnStart(ctx context.Context, info TaskInfo) {
	p.render(ctx, info, true)
}

func (p *Progress) OnProgress(ctx context.Context, info TaskInfo) {
	p.render(ctx, info, false)
}

func (p *Progress) OnStateChange(ctx context.Context, info TaskInfo) {
	p.render(ctx, info, true)
}

func (p *Progress) OnUploadStart(ctx context.Context, info TaskInfo, _ int64) {
	p.render(ctx, info, true)
}

func (p *Progress) OnUploadProgress(ctx context.Context, info TaskInfo, _, _ int64) {
	p.render(ctx, info, false)
}

func (p *Progress) render(ctx context.Context, info TaskInfo, priority bool) {
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	if p.done {
		return
	}
	now := time.Now()
	if !priority && !p.lastUpdateAt.IsZero() && now.Sub(p.lastUpdateAt) < progressRenderInterval {
		return
	}
	message := buildBatchProgressMessage(info, p.skippedFiles, visibleActiveItems())
	if message.Err != nil {
		log.FromContext(ctx).Errorf("Failed to render batch progress message: %v", message.Err)
		return
	}
	if message.Text == p.lastText {
		return
	}
	p.lastText = message.Text
	p.lastUpdateAt = now
	p.editMessage(ctx, info.TaskID(), message, true)
}

func (p *Progress) OnDone(ctx context.Context, info TaskInfo, err error) {
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	if p.done {
		return
	}
	p.done = true
	message := buildBatchDoneMessage(info, p.skippedFiles, err)
	if message.Err != nil {
		log.FromContext(ctx).Errorf("Failed to render final batch progress message: %v", message.Err)
		return
	}
	p.lastText = message.Text
	p.editMessage(ctx, info.TaskID(), message, false)
}

func (p *Progress) editMessage(ctx context.Context, taskID string, message renderedBatchMessage, cancellable bool) {
	if message.Err != nil {
		log.FromContext(ctx).Errorf("Failed to render batch progress message: %v", message.Err)
		return
	}
	req := buildBatchEditMessageRequest(p.MessageID, taskID, message, cancellable)
	if ext := tgutil.ExtFromContext(ctx); ext != nil {
		if _, err := ext.EditMessage(p.ChatID, req); err != nil {
			log.FromContext(ctx).Errorf("Failed to edit batch progress message: %v", err)
		}
	}
}

func buildBatchEditMessageRequest(messageID int, taskID string, message renderedBatchMessage, cancellable bool) *tg.MessagesEditMessageRequest {
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

func buildBatchProgressText(info TaskInfo, skipped []string, activeLimit int) string {
	return buildBatchProgressMessage(info, skipped, activeLimit).Text
}

func buildBatchProgressMessage(info TaskInfo, skipped []string, activeLimit int) renderedBatchMessage {
	items := info.Items()
	completed, waiting, downloaded, failed := itemCounts(items)
	downloadSpeed, uploadSpeed := aggregateSpeeds(items)
	if activeLimit < 1 {
		activeLimit = 1
	}

	total := len(items) + len(skipped)
	downloadSpeedText := formatSpeed(downloadSpeed)
	uploadSpeedText := formatSpeed(uploadSpeed)
	header := localizedProgressMarkup(i18nk.BotMsgProgressBatchStatusHeader, map[string]any{
		"Total":         total,
		"Completed":     completed,
		"Downloaded":    downloaded,
		"Waiting":       waiting,
		"DownloadSpeed": downloadSpeedText,
		"UploadSpeed":   uploadSpeedText,
	})

	var markup strings.Builder
	markup.WriteString(header)

	visibleItems, hiddenTransfers, summarizedConfirming := visibleBatchItems(items, activeLimit)
	for _, item := range visibleItems {
		markup.WriteString("\n\n")
		markup.WriteString(formatActiveItemMarkup(item, len(items)))
	}

	if hiddenTransfers > 0 {
		markup.WriteString("\n\n")
		markup.WriteString(localizedProgressMarkup(i18nk.BotMsgProgressBatchSummaryHiddenActive, map[string]any{"Count": hiddenTransfers}))
	}
	if summarizedConfirming > 0 {
		markup.WriteString("\n\n")
		markup.WriteString(localizedProgressMarkup(i18nk.BotMsgProgressBatchSummaryConfirming, map[string]any{"Count": summarizedConfirming}))
	}
	if failed > 0 {
		markup.WriteString("\n")
		markup.WriteString(localizedProgressMarkup(i18nk.BotMsgProgressBatchSummaryFailed, map[string]any{"Count": failed}))
	}
	if len(skipped) > 0 {
		markup.WriteString("\n")
		markup.WriteString(localizedProgressMarkup(i18nk.BotMsgProgressBatchSummarySkipped, map[string]any{"Count": len(skipped)}))
	}
	return completeBatchMessage(markup.String())
}

func buildBatchDoneMarkup(info TaskInfo, skipped []string, err error) string {
	items := info.Items()
	totalSize := info.ActualTotalSize()
	if totalSize == 0 {
		totalSize = info.TotalSize()
	}
	if err == nil {
		if len(skipped) > 0 {
			return localizedProgressMarkup(i18nk.BotMsgProgressBatchDoneWithSkipped, map[string]any{
				"Success": len(items),
				"Skipped": len(skipped),
				"Size":    dlutil.FormatSize(totalSize),
			})
		}
		return localizedProgressMarkup(i18nk.BotMsgProgressBatchDone, map[string]any{
			"Count": len(items),
			"Size":  dlutil.FormatSize(totalSize),
		})
	}
	completed, _, _, failed := itemCounts(items)
	incomplete := max(len(items)-completed-failed, 0)
	if errors.Is(err, context.Canceled) {
		return localizedProgressMarkup(i18nk.BotMsgProgressBatchCanceled, map[string]any{
			"Total":      len(items) + len(skipped),
			"Completed":  completed,
			"Incomplete": incomplete,
			"Skipped":    len(skipped),
		})
	}

	failedItems := make([]TaskItemProgress, 0, failed)
	for _, item := range items {
		if item.Phase == ItemPhaseFailed {
			failedItems = append(failedItems, item)
		}
	}
	if len(failedItems) > 1 && failedItems[0].FailureStage == FailureStageBatchUpload {
		return localizedProgressMarkup(i18nk.BotMsgProgressBatchFailedGroup, map[string]any{
			"Affected":   len(failedItems),
			"Reason":     displayError(firstError(failedItems), err),
			"Completed":  completed,
			"Failed":     len(failedItems),
			"Incomplete": incomplete,
		})
	}
	if len(failedItems) == 0 {
		return localizedProgressMarkup(i18nk.BotMsgProgressBatchFailedTask, map[string]any{
			"Reason":     displayError("", err),
			"Completed":  completed,
			"Incomplete": incomplete,
		})
	}
	item := failedItems[0]
	return localizedProgressMarkup(i18nk.BotMsgProgressBatchFailedItem, map[string]any{
		"Index":      item.Index,
		"Name":       truncateFilename(item.Name, maxDisplayNameRunes),
		"Stage":      failureStageLabel(item.FailureStage),
		"Progress":   failureProgress(item),
		"Speed":      failureSpeed(item),
		"Reason":     displayError(item.Error, err),
		"Completed":  completed,
		"Failed":     failed,
		"Incomplete": incomplete,
	})
}

func buildBatchDoneMessage(info TaskInfo, skipped []string, err error) renderedBatchMessage {
	return completeBatchMessage(buildBatchDoneMarkup(info, skipped, err))
}

func formatActiveItemMarkup(item TaskItemProgress, total int) string {
	data := map[string]any{
		"Index":    item.Index,
		"Total":    total,
		"Name":     truncateFilename(item.Name, maxDisplayNameRunes),
		"Speed":    formatSpeed(itemSpeed(item)),
		"Progress": itemPercent(item),
		"Bar":      textProgressBar(itemPercent(item)),
		"Current":  dlutil.FormatSize(itemBytes(item)),
		"Size":     dlutil.FormatSize(item.Size),
		"Attempt":  min(max(item.RetryAttempt, 1), max(item.RetryLimit, 1)),
		"Limit":    max(item.RetryLimit, 1),
		"Reason":   truncateRunes(item.Error, maxDisplayErrorRunes),
	}
	switch item.Phase {
	case ItemPhaseDownloading:
		if item.Size <= 0 {
			return localizedProgressMarkup(i18nk.BotMsgProgressBatchItemDownloadingUnknown, data)
		}
		return localizedProgressMarkup(i18nk.BotMsgProgressBatchItemDownloading, data)
	case ItemPhaseTransferring:
		if item.Size <= 0 {
			return localizedProgressMarkup(i18nk.BotMsgProgressBatchItemTransferringUnknown, data)
		}
		return localizedProgressMarkup(i18nk.BotMsgProgressBatchItemTransferring, data)
	case ItemPhaseUploading:
		return localizedProgressMarkup(i18nk.BotMsgProgressBatchItemUploading, data)
	case ItemPhaseRetrying:
		return localizedProgressMarkup(i18nk.BotMsgProgressBatchItemRetrying, data)
	case ItemPhaseConfirming:
		return localizedProgressMarkup(i18nk.BotMsgProgressBatchItemConfirming, data)
	default:
		return ""
	}
}

func localizedProgressMarkup(key i18nk.Key, data map[string]any) string {
	return i18n.T(key, tgutil.EscapeHTMLTemplateData(data))
}

func visibleBatchItems(items []TaskItemProgress, limit int) (visible []TaskItemProgress, hiddenTransfers, summarizedConfirming int) {
	visible = make([]TaskItemProgress, 0, limit)
	transferCount := 0
	confirmingCount := 0
	for _, item := range items {
		switch {
		case isTransferPhase(item.Phase):
			transferCount++
			if len(visible) < limit {
				visible = append(visible, item)
			}
		case item.Phase == ItemPhaseConfirming:
			confirmingCount++
		}
	}
	hiddenTransfers = transferCount - len(visible)
	if confirmingCount == 1 && len(visible) < limit {
		for _, item := range items {
			if item.Phase == ItemPhaseConfirming {
				visible = append(visible, item)
				return visible, hiddenTransfers, 0
			}
		}
	}
	return visible, hiddenTransfers, confirmingCount
}

func completeBatchMessage(markup string) renderedBatchMessage {
	text, entities, err := tgutil.RenderHTML(markup)
	return renderedBatchMessage{Text: text, Entities: entities, Err: err}
}

func itemCounts(items []TaskItemProgress) (completed, waiting, downloaded, failed int) {
	for _, item := range items {
		switch item.Phase {
		case ItemPhaseCompleted:
			completed++
		case ItemPhaseWaiting:
			waiting++
		case ItemPhaseDownloaded:
			downloaded++
		case ItemPhaseFailed:
			failed++
		}
	}
	return
}

func aggregateSpeeds(items []TaskItemProgress) (download, upload float64) {
	for _, item := range items {
		switch item.Phase {
		case ItemPhaseDownloading:
			download += item.DownloadSpeed
		case ItemPhaseTransferring:
			download += item.DownloadSpeed
			upload += item.UploadSpeed
		case ItemPhaseUploading:
			upload += item.UploadSpeed
		}
	}
	return
}

func isTransferPhase(phase ItemPhase) bool {
	switch phase {
	case ItemPhaseDownloading, ItemPhaseTransferring, ItemPhaseUploading, ItemPhaseRetrying:
		return true
	default:
		return false
	}
}

func itemBytes(item TaskItemProgress) int64 {
	switch item.Phase {
	case ItemPhaseDownloading, ItemPhaseTransferring:
		return item.Downloaded
	default:
		return item.Uploaded
	}
}

func itemSpeed(item TaskItemProgress) float64 {
	switch item.Phase {
	case ItemPhaseDownloading, ItemPhaseTransferring:
		return item.DownloadSpeed
	case ItemPhaseUploading:
		return item.UploadSpeed
	case ItemPhaseRetrying:
		return item.UploadSpeed
	default:
		return 0
	}
}

func itemPercent(item TaskItemProgress) int {
	if item.Size <= 0 {
		return 0
	}
	return int(min(itemBytes(item), item.Size) * 100 / item.Size)
}

func textProgressBar(percent int) string {
	percent = min(max(percent, 0), 100)
	filled := percent * progressBarWidth / 100
	return strings.Repeat("🟩", filled) + strings.Repeat("⬜️", progressBarWidth-filled)
}

func formatSpeed(speed float64) string {
	if speed <= 0 {
		return "0 B/s"
	}
	return dlutil.FormatSize(int64(speed)) + "/s"
}

func truncateFilename(name string, limit int) string {
	if utf8.RuneCountInString(name) <= limit {
		return name
	}
	ext := path.Ext(name)
	if utf8.RuneCountInString(ext) >= limit-2 {
		return truncateRunes(name, limit-1) + "…"
	}
	base := strings.TrimSuffix(name, ext)
	baseLimit := limit - utf8.RuneCountInString(ext) - 1
	return truncateRunes(base, baseLimit) + "…" + ext
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func displayError(itemError string, fallback error) string {
	if itemError == "" && fallback != nil {
		itemError = compactError(fallback)
	}
	return truncateRunes(itemError, maxDisplayErrorRunes)
}

func firstError(items []TaskItemProgress) string {
	for _, item := range items {
		if item.Error != "" {
			return item.Error
		}
	}
	return ""
}

func failureStageLabel(stage FailureStage) string {
	switch stage {
	case FailureStageDownload:
		return i18n.T(i18nk.BotMsgProgressBatchFailureStageDownload, nil)
	case FailureStageCache:
		return i18n.T(i18nk.BotMsgProgressBatchFailureStageCache, nil)
	case FailureStageUpload:
		return i18n.T(i18nk.BotMsgProgressBatchFailureStageUpload, nil)
	case FailureStageConfirm:
		return i18n.T(i18nk.BotMsgProgressBatchFailureStageConfirm, nil)
	case FailureStageBatchUpload:
		return i18n.T(i18nk.BotMsgProgressBatchFailureStageBatchUpload, nil)
	default:
		return i18n.T(i18nk.BotMsgProgressBatchFailureStageInternal, nil)
	}
}

func failureProgress(item TaskItemProgress) string {
	if item.Size <= 0 {
		return dlutil.FormatSize(failureBytes(item))
	}
	return fmt.Sprintf("%d%%", min(failureBytes(item), item.Size)*100/item.Size)
}

func failureSpeed(item TaskItemProgress) string {
	if item.FailureStage == FailureStageDownload || item.FailureStage == FailureStageCache {
		return formatSpeed(item.DownloadSpeed)
	}
	return formatSpeed(item.UploadSpeed)
}

func failureBytes(item TaskItemProgress) int64 {
	if item.FailureStage == FailureStageDownload || item.FailureStage == FailureStageCache {
		return item.Downloaded
	}
	return item.Uploaded
}

func visibleActiveItems() int {
	return min(max(config.C().Workers, 1), maxVisibleActiveItems)
}

func NewProgressTracker(messageID int, chatID int64) ProgressTracker {
	return NewProgressTrackerWithSkipped(messageID, chatID, nil)
}

func NewProgressTrackerWithSkipped(messageID int, chatID int64, skippedFiles []string) ProgressTracker {
	return &Progress{
		MessageID:    messageID,
		ChatID:       chatID,
		skippedFiles: skippedFiles,
	}
}
