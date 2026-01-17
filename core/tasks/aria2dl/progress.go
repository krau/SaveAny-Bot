package aria2dl

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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
	"github.com/krau/SaveAny-Bot/pkg/aria2"
)

type ProgressTracker interface {
	OnStart(ctx context.Context, task *Task)
	OnProgress(ctx context.Context, task *Task, status *aria2.Status)
	OnDone(ctx context.Context, task *Task, err error)
}

type Progress struct {
	msgID             int
	chatID            int64
	start             time.Time
	lastUpdatePercent atomic.Int32
}

// OnStart implements ProgressTracker.
func (p *Progress) OnStart(ctx context.Context, task *Task) {
	logger := log.FromContext(ctx)
	p.start = time.Now()
	p.lastUpdatePercent.Store(0)
	logger.Infof("Aria2 task started: message_id=%d, chat_id=%d, gid=%s", p.msgID, p.chatID, task.GID)
	ext := tgutil.ExtFromContext(ctx)
	if ext == nil {
		return
	}
	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressAria2Start, map[string]any{
			"GID": task.GID,
		}))); err != nil {
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
func (p *Progress) OnProgress(ctx context.Context, task *Task, status *aria2.Status) {
	totalLength, _ := strconv.ParseInt(status.TotalLength, 10, 64)
	completedLength, _ := strconv.ParseInt(status.CompletedLength, 10, 64)
	downloadSpeed, _ := strconv.ParseInt(status.DownloadSpeed, 10, 64)

	if totalLength == 0 {
		return
	}

	percent := int((completedLength * 100) / totalLength)
	if p.lastUpdatePercent.Load() == int32(percent) {
		return
	}
	p.lastUpdatePercent.Store(int32(percent))

	log.FromContext(ctx).Debugf("Aria2 progress update: %s, %d/%d", task.GID, completedLength, totalLength)

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressAria2Downloading, map[string]any{
			"GID": task.GID,
		})),
		styling.Plain(i18n.T(i18nk.BotMsgProgressDownloadedPrefix, nil)),
		styling.Code(fmt.Sprintf("%.2f MB / %.2f MB", float64(completedLength)/(1024*1024), float64(totalLength)/(1024*1024))),
		styling.Plain(i18n.T(i18nk.BotMsgProgressCurrentSpeedPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f MB/s", float64(downloadSpeed)/(1024*1024))),
		styling.Plain(i18n.T(i18nk.BotMsgProgressAvgSpeedPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f MB/s", dlutil.GetSpeed(completedLength, p.start)/(1024*1024))),
		styling.Plain(i18n.T(i18nk.BotMsgProgressCurrentProgressPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f%%", float64(percent))),
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
			logger.Infof("Aria2 task %s was canceled", task.TaskID())
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
			logger.Errorf("Aria2 task %s failed: %s", task.TaskID(), err)
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
	logger.Infof("Aria2 task %s completed successfully", task.TaskID())

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressAria2Done, map[string]any{
			"GID": task.GID,
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
		msgID:  msgID,
		chatID: userID,
	}
}
