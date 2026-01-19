package transfer

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

type ProgressTracker interface {
	OnStart(ctx context.Context, info TaskInfo)
	OnProgress(ctx context.Context, info TaskInfo)
	OnDone(ctx context.Context, info TaskInfo, err error)
}

type Progress struct {
	MessageID         int
	ChatID            int64
	start             time.Time
	lastUpdatePercent atomic.Int32
}

func NewProgressTracker(messageID int, chatID int64) ProgressTracker {
	return &Progress{
		MessageID: messageID,
		ChatID:    chatID,
	}
}

func (p *Progress) OnStart(ctx context.Context, info TaskInfo) {
	p.start = time.Now()
	p.lastUpdatePercent.Store(0)
	log.FromContext(ctx).Debugf("Transfer task progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)

	sizeMB := float64(info.TotalSize()) / (1024 * 1024)
	statsText := i18n.T(i18nk.BotMsgTransferStartStats, map[string]any{
		"SizeMB": fmt.Sprintf("%.2f", sizeMB),
		"Count":  info.Count(),
	})

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressTransferStartPrefix, nil)),
		styling.Code(statsText),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entities: %s", err)
		return
	}

	text, entities := entityBuilder.Complete()
	req := &tg.MessagesEditMessageRequest{
		ID: p.MessageID,
	}
	req.SetMessage(text)
	req.SetEntities(entities)
	req.SetReplyMarkup(&tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					tgutil.BuildCancelButton(info.TaskID()),
				},
			},
		},
	})

	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		_, err := ext.EditMessage(p.ChatID, req)
		if err != nil {
			log.FromContext(ctx).Errorf("Failed to send progress start message: %s", err)
		}
	}
}

func (p *Progress) OnProgress(ctx context.Context, info TaskInfo) {
	if !shouldUpdateProgress(info.TotalSize(), info.Uploaded(), int(p.lastUpdatePercent.Load())) {
		return
	}
	percent := int((info.Uploaded() * 100) / info.TotalSize())
	if p.lastUpdatePercent.Load() == int32(percent) {
		return
	}
	p.lastUpdatePercent.Store(int32(percent))

	log.FromContext(ctx).Debugf("Progress update: %s, %d/%d", info.TaskID(), info.Uploaded(), info.TotalSize())

	entityBuilder := entity.Builder{}
	var progressText strings.Builder

	progressText.WriteString(i18n.T(i18nk.BotMsgProgressTransferProgressPrefix, nil))
	fmt.Fprintf(&progressText, "%d%%", percent)
	progressText.WriteString(i18n.T(i18nk.BotMsgProgressTransferUploadedPrefix, nil))
	fmt.Fprintf(&progressText, "%.2f MB / %.2f MB",
		float64(info.Uploaded())/(1024*1024),
		float64(info.TotalSize())/(1024*1024))

	if p.start.Unix() > 0 {
		elapsed := time.Since(p.start)
		speed := float64(info.Uploaded()) / elapsed.Seconds()
		progressText.WriteString(i18n.T(i18nk.BotMsgProgressTransferSpeedPrefix, nil))
		progressText.WriteString(dlutil.FormatSize(int64(speed)) + "/s")

		if info.Uploaded() > 0 {
			remaining := time.Duration(float64(info.TotalSize()-info.Uploaded()) / speed * float64(time.Second))
			progressText.WriteString(i18n.T(i18nk.BotMsgProgressTransferRemainingTimePrefix, nil))
			progressText.WriteString(formatDuration(remaining))
		}
	}

	processing := info.Processing()
	if len(processing) > 0 {
		progressText.WriteString(i18n.T(i18nk.BotMsgProgressTransferProcessingPrefix, nil))
		for i, elem := range processing {
			if i >= 3 {
				progressText.WriteString(i18n.T(i18nk.BotMsgProgressTransferProcessingMore, map[string]any{"Count": len(processing) - 3}))
				break
			}
			fmt.Fprintf(&progressText, "- %s\n", elem.FileName())
		}
	}

	if err := styling.Perform(&entityBuilder,
		styling.Plain(progressText.String()),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entities: %s", err)
		return
	}

	text, entities := entityBuilder.Complete()
	req := &tg.MessagesEditMessageRequest{
		ID: p.MessageID,
	}
	req.SetMessage(text)
	req.SetEntities(entities)
	req.SetReplyMarkup(&tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					tgutil.BuildCancelButton(info.TaskID()),
				},
			},
		},
	})

	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		ext.EditMessage(p.ChatID, req)
	}
}

func (p *Progress) OnDone(ctx context.Context, info TaskInfo, err error) {
	log.FromContext(ctx).Debugf("Transfer task progress tracking done for message %d in chat %d", p.MessageID, p.ChatID)

	entityBuilder := entity.Builder{}
	var resultText strings.Builder

	if err != nil {
		resultText.WriteString(i18n.T(i18nk.BotMsgProgressTransferFailedPrefix, nil))
		resultText.WriteString(i18n.T(i18nk.BotMsgProgressErrorPrefix, nil))
		fmt.Fprintf(&resultText, "%v\n", err)
	} else {
		resultText.WriteString(i18n.T(i18nk.BotMsgProgressTransferSuccessPrefix, nil))
	}

	elapsed := time.Since(p.start)
	resultText.WriteString(i18n.T(i18nk.BotMsgProgressTransferTotalFilesPrefix, nil))
	fmt.Fprintf(&resultText, "%d\n", info.Count())
	resultText.WriteString(i18n.T(i18nk.BotMsgProgressTransferTotalSizePrefix, nil))
	fmt.Fprintf(&resultText, "%.2f MB\n", float64(info.TotalSize())/(1024*1024))
	resultText.WriteString(i18n.T(i18nk.BotMsgProgressTransferUploadedPrefix, nil))
	fmt.Fprintf(&resultText, "%.2f MB\n", float64(info.Uploaded())/(1024*1024))
	resultText.WriteString(i18n.T(i18nk.BotMsgProgressTransferElapsedTimePrefix, nil))
	fmt.Fprintf(&resultText, "%s\n", formatDuration(elapsed))

	if elapsed.Seconds() > 0 {
		avgSpeed := float64(info.Uploaded()) / elapsed.Seconds()
		resultText.WriteString(i18n.T(i18nk.BotMsgProgressTransferAvgSpeedPrefix, nil))
		fmt.Fprintf(&resultText, "%s/s\n", dlutil.FormatSize(int64(avgSpeed)))
	}

	failedFiles := info.FailedFiles()
	if len(failedFiles) > 0 {
		resultText.WriteString(i18n.T(i18nk.BotMsgProgressTransferFailedFilesPrefix, nil))
		fmt.Fprintf(&resultText, "%d\n", len(failedFiles))
		for i, name := range failedFiles {
			if i >= 5 {
				resultText.WriteString(i18n.T(i18nk.BotMsgProgressTransferProcessingMore, map[string]any{"Count": len(failedFiles) - 5}))
				break
			}
			fmt.Fprintf(&resultText, "- %s\n", name)
		}
	}

	if err := styling.Perform(&entityBuilder,
		styling.Plain(resultText.String()),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entities: %s", err)
		return
	}

	text, entities := entityBuilder.Complete()
	req := &tg.MessagesEditMessageRequest{
		ID: p.MessageID,
	}
	req.SetMessage(text)
	req.SetEntities(entities)

	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		ext.EditMessage(p.ChatID, req)
	}
}

func shouldUpdateProgress(total, current int64, lastPercent int) bool {
	if total == 0 {
		return false
	}
	currentPercent := int((current * 100) / total)
	return currentPercent > lastPercent && currentPercent%5 == 0
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
