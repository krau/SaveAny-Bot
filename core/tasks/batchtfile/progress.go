package batchtfile

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"
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
	startAt           atomic.Int64
	lastUpdatePercent atomic.Int32
	lastUpdateAt      atomic.Int64
	uploadStarted     atomic.Bool
	updateMu          sync.Mutex
	skippedFiles      []string
}

const (
	uploadProgressMinInterval = time.Second
	uploadProgressMaxInterval = 3 * time.Second
)

func (p *Progress) OnStart(ctx context.Context, info TaskInfo) {
	p.startAt.Store(time.Now().UnixNano())
	p.lastUpdatePercent.Store(0)
	p.lastUpdateAt.Store(0)
	p.uploadStarted.Store(false)
	log.FromContext(ctx).Debugf("Batch task progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressBatchStartPrefix, nil)),
		styling.Code(batchProgressSummary(info.TotalSize(), info.Count())),
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
		}},
	)
	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		ext.EditMessage(p.ChatID, req)
		return
	}
}

func (p *Progress) OnProgress(ctx context.Context, info TaskInfo) {
	if p.uploadStarted.Load() {
		return
	}
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	if p.uploadStarted.Load() {
		return
	}
	downloaded := min(info.Downloaded(), info.TotalSize())
	if !shouldUpdateProgress(info.TotalSize(), downloaded, int(p.lastUpdatePercent.Load())) {
		return
	}
	percent := int((downloaded * 100) / info.TotalSize())
	if p.lastUpdatePercent.Load() == int32(percent) {
		return
	}
	p.lastUpdatePercent.Store(int32(percent))
	log.FromContext(ctx).Debugf("Progress update: %s, %d/%d", info.TaskID(), downloaded, info.TotalSize())
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressBatchProcessingPrefix, nil)),
		styling.Code(batchProgressSummary(info.TotalSize(), info.Count())),
		styling.Plain(i18n.T(i18nk.BotMsgProgressProcessingListPrefix, nil)),
		func() styling.StyledTextOption {
			var lines []string
			for _, elem := range info.Processing() {
				lines = append(lines, fmt.Sprintf("  - %s (%.2f MB)", elem.FileName(), float64(elem.FileSize())/(1024*1024)))
			}
			if len(lines) == 0 {
				lines = append(lines, i18n.T(i18nk.BotMsgProgressProcessingNone, nil))
			}
			return styling.Plain(slice.Join(lines, "\n"))
		}(),
		styling.Plain(i18n.T(i18nk.BotMsgProgressAvgSpeedPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f MB/s", dlutil.GetSpeed(downloaded, time.Unix(0, p.startAt.Load()))/(1024*1024))),
		styling.Plain(i18n.T(i18nk.BotMsgProgressCurrentProgressPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f%%", float64(downloaded)/float64(info.TotalSize())*100)),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entities: %s", err)
		return
	}
	text, entities := entityBuilder.Complete()
	if p.uploadStarted.Load() {
		return
	}
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
		}},
	)
	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		ext.EditMessage(p.ChatID, req)
		return
	}
}

func (p *Progress) OnUploadStart(ctx context.Context, info TaskInfo, total int64) {
	p.uploadStarted.Store(true)
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	start := time.Now()
	p.startAt.Store(start.UnixNano())
	p.lastUpdatePercent.Store(0)
	p.lastUpdateAt.Store(start.UnixNano())
	log.FromContext(ctx).Debugf("Batch upload progress tracking started: %s", info.TaskID())

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressBatchUploadingPrefix, nil)),
		styling.Code(batchProgressSummary(total, info.Count())),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build batch upload entities: %s", err)
		return
	}

	text, entities := entityBuilder.Complete()
	req := &tg.MessagesEditMessageRequest{ID: p.MessageID}
	req.SetMessage(text)
	req.SetEntities(entities)
	req.SetReplyMarkup(&tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{{
			Buttons: []tg.KeyboardButtonClass{tgutil.BuildCancelButton(info.TaskID())},
		}},
	})
	if ext := tgutil.ExtFromContext(ctx); ext != nil {
		ext.EditMessage(p.ChatID, req)
	}
}

func (p *Progress) OnUploadProgress(ctx context.Context, info TaskInfo, uploaded, total int64) {
	if total <= 0 || uploaded <= 0 {
		return
	}
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	uploaded = min(uploaded, total)
	now := time.Now()
	lastUpdateAt := time.Unix(0, p.lastUpdateAt.Load())
	lastPercent := int(p.lastUpdatePercent.Load())
	if !shouldUpdateUploadProgress(total, uploaded, lastPercent, now.Sub(lastUpdateAt)) {
		return
	}

	percent := int32((uploaded * 100) / total)
	p.lastUpdatePercent.Store(percent)
	p.lastUpdateAt.Store(now.UnixNano())
	log.FromContext(ctx).Debugf("Batch upload progress update: %s, %d/%d", info.TaskID(), uploaded, total)

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressBatchUploadingPrefix, nil)),
		styling.Code(batchProgressSummary(total, info.Count())),
		styling.Plain(i18n.T(i18nk.BotMsgProgressAvgSpeedPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f MB/s", dlutil.GetSpeed(uploaded, time.Unix(0, p.startAt.Load()))/(1024*1024))),
		styling.Plain(i18n.T(i18nk.BotMsgProgressCurrentProgressPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f%%", float64(uploaded)/float64(total)*100)),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build batch upload entities: %s", err)
		return
	}

	text, entities := entityBuilder.Complete()
	req := &tg.MessagesEditMessageRequest{ID: p.MessageID}
	req.SetMessage(text)
	req.SetEntities(entities)
	req.SetReplyMarkup(&tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{{
			Buttons: []tg.KeyboardButtonClass{tgutil.BuildCancelButton(info.TaskID())},
		}},
	})
	if ext := tgutil.ExtFromContext(ctx); ext != nil {
		ext.EditMessage(p.ChatID, req)
	}
}

func shouldUpdateUploadProgress(total, uploaded int64, lastPercent int, elapsed time.Duration) bool {
	if total <= 0 || uploaded <= 0 {
		return false
	}
	if uploaded >= total {
		return lastPercent < 100 && elapsed >= uploadProgressMinInterval
	}
	if elapsed < uploadProgressMinInterval {
		return false
	}
	return shouldUpdateProgress(total, uploaded, lastPercent) || elapsed >= uploadProgressMaxInterval
}

func batchProgressSummary(total int64, count int) string {
	return i18n.T(i18nk.BotMsgProgressBatchSummary, map[string]any{
		"SizeMB": fmt.Sprintf("%.2f", float64(total)/(1024*1024)),
		"Count":  count,
	})
}

func (p *Progress) OnDone(ctx context.Context, info TaskInfo, err error) {
	if err != nil {
		log.FromContext(ctx).Errorf("Batch task %s failed: %s", info.TaskID(), err)
	} else {
		log.FromContext(ctx).Debugf("Batch task %s completed successfully", info.TaskID())
	}
	entityBuilder := entity.Builder{}
	var stylingErr error

	if err != nil {
		if errors.Is(err, context.Canceled) {
			stylingErr = styling.Perform(&entityBuilder,
				styling.Plain(i18n.T(i18nk.BotMsgProgressTaskCanceled, nil)),
			)
		} else {
			stylingErr = styling.Perform(&entityBuilder,
				styling.Plain(i18n.T(i18nk.BotMsgProgressTaskFailedWithError, map[string]any{
					"Error": "",
				})),
				styling.Code(err.Error()),
			)
		}
	} else {
		stylingErr = styling.Perform(&entityBuilder,
			styling.Plain(i18n.T(i18nk.BotMsgProgressBatchDonePrefix, nil)),
			styling.Code(strconv.Itoa(info.Count())),
			styling.Plain(i18n.T(i18nk.BotMsgProgressTotalSizePrefix, nil)),
			styling.Code(fmt.Sprintf("%.2f MB", float64(info.TotalSize())/(1024*1024))),
			func() styling.StyledTextOption {
				if len(p.skippedFiles) == 0 {
					return styling.Plain("")
				}
				return styling.Plain("\n\n" + i18n.T(i18nk.BotMsgCommonInfoConflictFilesSkipped, map[string]any{
					"Skipped": strings.Join(p.skippedFiles, "\n"),
				}))
			}(),
		)
	}

	if stylingErr != nil {
		log.FromContext(ctx).Errorf("Failed to build entities: %s", stylingErr)
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
