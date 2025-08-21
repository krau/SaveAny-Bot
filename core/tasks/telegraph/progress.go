package telegraph

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

type ProgressTracker interface {
	OnStart(ctx context.Context, info TaskInfo)
	OnProgress(ctx context.Context, info TaskInfo)
	OnDone(ctx context.Context, info TaskInfo, err error)
}

type Progress struct {
	MessageID int
	ChatID    int64
}

func (p *Progress) OnStart(ctx context.Context, info TaskInfo) {
	logger := log.FromContext(ctx)
	logger.Debugf("Telegraph task progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain("开始下载Telegraph\n图片数量: "),
		styling.Code(fmt.Sprintf("%d", info.TotalPics())),
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
	if !shouldUpdateProgress(info.Downloaded(), int64(info.TotalPics())) {
		return
	}
	log.FromContext(ctx).Debugf("Progress update: %s, %d/%d", info.TaskID(), info.Downloaded(), info.TotalPics())
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain("正在下载\n当前进度: "),
		styling.Code(fmt.Sprintf("%d/%d", info.Downloaded(), info.TotalPics())),
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

func (p *Progress) OnDone(ctx context.Context, info TaskInfo, err error) {
	logger := log.FromContext(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Infof("Telegraph task %s was canceled", info.TaskID())
			ext := tgutil.ExtFromContext(ctx)
			if ext != nil {
				ext.EditMessage(p.ChatID, &tg.MessagesEditMessageRequest{
					ID:      p.MessageID,
					Message: fmt.Sprintf("处理已取消: %s", info.TaskID()),
				})
			}
		} else {
			logger.Errorf("Telegraph task %s failed: %s", info.TaskID(), err)
			ext := tgutil.ExtFromContext(ctx)
			if ext != nil {
				ext.EditMessage(p.ChatID, &tg.MessagesEditMessageRequest{
					ID:      p.MessageID,
					Message: fmt.Sprintf("处理失败: %s", err.Error()),
				})
			}
		}
		return
	}
	logger.Infof("Telegraph task %s completed successfully", info.TaskID())

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain("处理完成\n图片数量: "),
		styling.Code(fmt.Sprintf("%d", info.TotalPics())),
		styling.Plain("\n保存路径: "),
		styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
	); err != nil {
		logger.Errorf("Failed to build entities: %s", err)
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

func NewProgress(messageID int, chatID int64) *Progress {
	return &Progress{
		MessageID: messageID,
		ChatID:    chatID,
	}
}
