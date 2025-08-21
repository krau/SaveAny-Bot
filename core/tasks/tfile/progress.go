package tfile

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
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

type ProgressTracker interface {
	OnStart(ctx context.Context, info TaskInfo)
	OnProgress(ctx context.Context, info TaskInfo, downloaded, total int64)
	OnDone(ctx context.Context, info TaskInfo, err error)
}

type Progress struct {
	MessageID         int
	ChatID            int64
	start             time.Time
	lastUpdatePercent atomic.Int32
}

func (p *Progress) OnStart(ctx context.Context, info TaskInfo) {
	p.start = time.Now()
	p.lastUpdatePercent.Store(0)
	log.FromContext(ctx).Debugf("Progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain("开始下载\n文件名: "),
		styling.Code(info.FileName()),
		styling.Plain("\n保存路径: "),
		styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
		styling.Plain("\n文件大小: "),
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
		styling.Plain("正在处理下载任务\n文件名: "),
		styling.Code(info.FileName()),
		styling.Plain("\n保存路径: "),
		styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
		styling.Plain("\n文件大小: "),
		styling.Code(fmt.Sprintf("%.2f MB", float64(total)/(1024*1024))),
		styling.Plain("\n平均速度: "),
		styling.Bold(fmt.Sprintf("%.2f MB/s", dlutil.GetSpeed(downloaded, p.start)/(1024*1024))),
		styling.Plain("\n当前进度: "),
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
				styling.Plain("任务已取消\n文件名: "),
				styling.Code(info.FileName()),
			)
		} else {
			stylingErr = styling.Perform(&entityBuilder,
				styling.Plain("下载失败\n文件名: "),
				styling.Code(info.FileName()),
				styling.Plain("\n错误: "),
				styling.Bold(err.Error()),
			)
		}
	} else {
		stylingErr = styling.Perform(&entityBuilder,
			styling.Plain("下载完成\n文件名: "),
			styling.Code(info.FileName()),
			styling.Plain("\n保存路径: "),
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
