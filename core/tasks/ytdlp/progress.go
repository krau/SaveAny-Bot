package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"

	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

// ProgressTracker defines the interface for tracking ytdlp task progress
type ProgressTracker interface {
	OnStart(ctx context.Context, task *Task)
	OnProgress(ctx context.Context, task *Task, status string)
	OnDone(ctx context.Context, task *Task, err error)
}

type Progress struct {
	msgID             int
	chatID            int64
	start             time.Time
	lastUpdate        atomic.Value // stores time.Time
	minUpdateInterval time.Duration
}

// OnStart implements ProgressTracker.
func (p *Progress) OnStart(ctx context.Context, task *Task) {
	logger := log.FromContext(ctx)
	p.start = time.Now()
	p.lastUpdate.Store(time.Now())
	p.minUpdateInterval = 2 * time.Second // Avoid too frequent updates
	logger.Infof("yt-dlp task started: message_id=%d, chat_id=%d, urls=%d", p.msgID, p.chatID, len(task.URLs))
	ext := tgutil.ExtFromContext(ctx)
	if ext == nil {
		return
	}
	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressYtdlpStart, map[string]any{
			"Count": len(task.URLs),
		})),
		styling.Plain(i18n.T(i18nk.BotMsgProgressSavePathPrefix, nil)),
		styling.Code(fmt.Sprintf("[%s]:%s", task.Storage.Name(), task.StorPath)),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entities: %s", err)
		return
	}
	text, entities := entityBuilder.Complete()
	req := &tg.MessagesEditMessageRequest{
		ID: p.msgID,
	}
	req.SetMessage(text)
	req.SetEntities(entities)
	req.SetReplyMarkup(&tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					tgutil.BuildCancelButton(task.TaskID()),
				},
			},
		}},
	)
	ext.EditMessage(p.chatID, req)
}

// OnProgress implements ProgressTracker.
func (p *Progress) OnProgress(ctx context.Context, task *Task, status string) {
	// Throttle updates to avoid flooding Telegram API
	lastUpdateTime := p.lastUpdate.Load().(time.Time)
	if time.Since(lastUpdateTime) < p.minUpdateInterval {
		return
	}
	p.lastUpdate.Store(time.Now())

	log.FromContext(ctx).Debugf("yt-dlp progress update: %s", status)

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressYtdlpDownloading, map[string]any{
			"Count": len(task.URLs),
		})),
		styling.Plain(i18n.T(i18nk.BotMsgProgressSavePathPrefix, nil)),
		styling.Code(fmt.Sprintf("[%s]:%s", task.Storage.Name(), task.StorPath)),
		styling.Plain("\n\n"),
		styling.Plain(status),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entities: %s", err)
		return
	}
	text, entities := entityBuilder.Complete()
	req := &tg.MessagesEditMessageRequest{
		ID: p.msgID,
	}
	req.SetMessage(text)
	req.SetEntities(entities)
	req.SetReplyMarkup(&tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					tgutil.BuildCancelButton(task.TaskID()),
				},
			},
		}},
	)
	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		ext.EditMessage(p.chatID, req)
	}
}

// OnDone implements ProgressTracker.
func (p *Progress) OnDone(ctx context.Context, task *Task, err error) {
	logger := log.FromContext(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Infof("yt-dlp task %s was canceled", task.TaskID())
			ext := tgutil.ExtFromContext(ctx)
			if ext != nil {
				ext.EditMessage(p.chatID, &tg.MessagesEditMessageRequest{
					ID: p.msgID,
					Message: i18n.T(i18nk.BotMsgProgressTaskCanceledWithId, map[string]any{
						"TaskID": task.TaskID(),
					}),
				})
			}
		} else {
			logger.Errorf("yt-dlp task %s failed: %s", task.TaskID(), err)
			ext := tgutil.ExtFromContext(ctx)
			if ext != nil {
				ext.EditMessage(p.chatID, &tg.MessagesEditMessageRequest{
					ID: p.msgID,
					Message: i18n.T(i18nk.BotMsgProgressTaskFailedWithError, map[string]any{
						"Error": err.Error(),
					}),
				})
			}
		}
		return
	}
	logger.Infof("yt-dlp task %s completed successfully", task.TaskID())

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressYtdlpDone, map[string]any{
			"Count": len(task.URLs),
		})),
		styling.Plain(i18n.T(i18nk.BotMsgProgressSavePathPrefix, nil)),
		styling.Code(fmt.Sprintf("[%s]:%s", task.Storage.Name(), task.StorPath)),
	); err != nil {
		logger.Errorf("Failed to build entities: %s", err)
		return
	}
	text, entities := entityBuilder.Complete()
	req := &tg.MessagesEditMessageRequest{
		ID: p.msgID,
	}
	req.SetMessage(text)
	req.SetEntities(entities)

	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		ext.EditMessage(p.chatID, req)
	}
}

var _ ProgressTracker = (*Progress)(nil)

func NewProgress(msgID int, userID int64) ProgressTracker {
	return &Progress{
		msgID:             msgID,
		chatID:            userID,
		minUpdateInterval: 2 * time.Second,
	}
}
