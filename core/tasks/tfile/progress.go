package tfile

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
}

const (
	uploadProgressMinInterval = time.Second
	uploadProgressMaxInterval = 3 * time.Second
)

func (p *Progress) OnStart(ctx context.Context, info TaskInfo) {
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	p.start = time.Now()
	p.lastUpdatePercent.Store(0)
	log.FromContext(ctx).Debugf("Progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressFileStartPrefix, nil)),
		styling.Code(info.FileName()),
		styling.Plain(i18n.T(i18nk.BotMsgProgressSavePathPrefix, nil)),
		styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
		styling.Plain(i18n.T(i18nk.BotMsgProgressFileSizePrefix, nil)),
		styling.Code(fmt.Sprintf("%.2f MB", float64(info.FileSize())/(1024*1024))),
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

func (p *Progress) OnProgress(ctx context.Context, info TaskInfo, downloaded, total int64) {
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	if !shouldUpdateProgress(total, downloaded, int(p.lastUpdatePercent.Load())) {
		return
	}
	percent := int32((downloaded * 100) / total)
	if p.lastUpdatePercent.Load() == percent {
		return
	}
	p.lastUpdatePercent.Store(percent)
	log.FromContext(ctx).Debugf("Progress update: %s, %d/%d", info.FileName(), downloaded, total)
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressFileProcessingPrefix, nil)),
		styling.Code(info.FileName()),
		styling.Plain(i18n.T(i18nk.BotMsgProgressSavePathPrefix, nil)),
		styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
		styling.Plain(i18n.T(i18nk.BotMsgProgressFileSizePrefix, nil)),
		styling.Code(fmt.Sprintf("%.2f MB", float64(total)/(1024*1024))),
		styling.Plain(i18n.T(i18nk.BotMsgProgressAvgSpeedPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f MB/s", dlutil.GetSpeed(downloaded, p.start)/(1024*1024))),
		styling.Plain(i18n.T(i18nk.BotMsgProgressCurrentProgressPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f%%", float64(downloaded)/float64(total)*100)),
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

func (p *Progress) OnUploadStart(ctx context.Context, info TaskInfo, total int64) {
	p.updateMu.Lock()
	defer p.updateMu.Unlock()
	p.start = time.Now()
	p.lastUpdatePercent.Store(0)
	p.lastUpdateAt.Store(p.start.UnixNano())
	log.FromContext(ctx).Debugf("Upload progress tracking started: %s", info.FileName())

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressUploadingPrefix, nil)),
		styling.Code(info.FileName()),
		styling.Plain(i18n.T(i18nk.BotMsgProgressSavePathPrefix, nil)),
		styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
		styling.Plain(i18n.T(i18nk.BotMsgProgressFileSizePrefix, nil)),
		styling.Code(fmt.Sprintf("%.2f MB", float64(total)/(1024*1024))),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build upload entities: %s", err)
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
	if uploaded > total {
		uploaded = total
	}

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

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressUploadingPrefix, nil)),
		styling.Code(info.FileName()),
		styling.Plain(i18n.T(i18nk.BotMsgProgressSavePathPrefix, nil)),
		styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
		styling.Plain(i18n.T(i18nk.BotMsgProgressFileSizePrefix, nil)),
		styling.Code(fmt.Sprintf("%.2f MB", float64(total)/(1024*1024))),
		styling.Plain(i18n.T(i18nk.BotMsgProgressAvgSpeedPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f MB/s", dlutil.GetSpeed(uploaded, p.start)/(1024*1024))),
		styling.Plain(i18n.T(i18nk.BotMsgProgressCurrentProgressPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f%%", float64(uploaded)/float64(total)*100)),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build upload entities: %s", err)
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

func (p *Progress) OnDone(ctx context.Context, info TaskInfo, err error) {
	if err != nil {
		log.FromContext(ctx).Errorf("Progress error for file [%s]: %v", info.FileName(), err)
	} else {
		log.FromContext(ctx).Debugf("Progress done for file [%s]", info.FileName())
	}

	entityBuilder := entity.Builder{}
	var stylingErr error

	if err != nil {
		if errors.Is(err, context.Canceled) {
			stylingErr = styling.Perform(&entityBuilder,
				styling.Plain(i18n.T(i18nk.BotMsgProgressTaskCanceled, nil)),
				styling.Plain("\n"),
				styling.Plain(i18n.T(i18nk.BotMsgProgressFileNamePrefix, nil)),
				styling.Code(info.FileName()),
			)
		} else {
			stylingErr = styling.Perform(&entityBuilder,
				styling.Plain(i18n.T(i18nk.BotMsgProgressFileFailedPrefix, nil)),
				styling.Code(info.FileName()),
				styling.Plain(i18n.T(i18nk.BotMsgProgressErrorPrefix, nil)),
				styling.Bold(err.Error()),
			)
		}
	} else {
		stylingErr = styling.Perform(&entityBuilder,
			styling.Plain(i18n.T(i18nk.BotMsgProgressFileDonePrefix, nil)),
			styling.Code(info.FileName()),
			styling.Plain(i18n.T(i18nk.BotMsgProgressSavePathPrefix, nil)),
			styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
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
