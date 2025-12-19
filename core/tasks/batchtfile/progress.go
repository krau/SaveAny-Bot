package batchtfile

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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
	start             time.Time
	lastUpdatePercent atomic.Int32
}

func (p *Progress) OnStart(ctx context.Context, info TaskInfo) {
	p.start = time.Now()
	p.lastUpdatePercent.Store(0)
	log.FromContext(ctx).Debugf("Batch task progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressBatchStartPrefix, nil)),
		styling.Code(fmt.Sprintf("%.2f MB (%d个文件)", float64(info.TotalSize())/(1024*1024), info.Count())),
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
	if !shouldUpdateProgress(info.TotalSize(), info.Downloaded(), int(p.lastUpdatePercent.Load())) {
		return
	}
	percent := int((info.Downloaded() * 100) / info.TotalSize())
	if p.lastUpdatePercent.Load() == int32(percent) {
		return
	}
	p.lastUpdatePercent.Store(int32(percent))
	log.FromContext(ctx).Debugf("Progress update: %s, %d/%d", info.TaskID(), info.Downloaded(), info.TotalSize())
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain(i18n.T(i18nk.BotMsgProgressBatchProcessingPrefix, nil)),
		styling.Code(fmt.Sprintf("%.2f MB (%d个文件)", float64(info.TotalSize())/(1024*1024), info.Count())),
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
		styling.Bold(fmt.Sprintf("%.2f MB/s", dlutil.GetSpeed(info.Downloaded(), p.start)/(1024*1024))),
		styling.Plain(i18n.T(i18nk.BotMsgProgressCurrentProgressPrefix, nil)),
		styling.Bold(fmt.Sprintf("%.2f%%", float64(info.Downloaded())/float64(info.TotalSize())*100)),
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
	return &Progress{
		MessageID: messageID,
		ChatID:    chatID,
	}
}
