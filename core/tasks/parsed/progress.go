package parsed

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

var progressUpdatesLevels = []struct {
	size        int64 // 文件大小阈值
	stepPercent int   // 每多少 % 更新一次
}{
	{10 << 20, 100},
	{50 << 20, 50},
	{200 << 20, 20},
	{500 << 20, 10},
}

func shouldUpdateProgress(total, downloaded int64, lastUpdatePercent int) bool {
	if total <= 0 || downloaded <= 0 {
		return false
	}

	percent := int((downloaded * 100) / total)
	if percent <= lastUpdatePercent {
		return false
	}

	step := progressUpdatesLevels[len(progressUpdatesLevels)-1].stepPercent
	for _, lvl := range progressUpdatesLevels {
		if total < lvl.size {
			step = lvl.stepPercent
			break
		}
	}

	return percent >= lastUpdatePercent+step
}

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
	logger := log.FromContext(ctx)
	p.start = time.Now()
	p.lastUpdatePercent.Store(0)
	logger.Debugf("Parsed task progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain(fmt.Sprintf("开始下载 %s 的资源\n总大小: ", info.Site())),
		styling.Code(fmt.Sprintf("%.2f MB (%d个资源)", float64(info.TotalBytes())/(1024*1024), info.TotalResources())),
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
	if !shouldUpdateProgress(info.TotalBytes(), info.DownloadedBytes(), int(p.lastUpdatePercent.Load())) {
		return
	}
	percent := int((info.DownloadedBytes() * 100) / info.TotalBytes())
	if p.lastUpdatePercent.Load() == int32(percent) {
		return
	}
	p.lastUpdatePercent.Store(int32(percent))
	log.FromContext(ctx).Debugf("Progress update: %s, %d/%d", info.TaskID(), info.DownloadedBytes(), info.TotalBytes())
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain("正在下载\n总大小: "),
		styling.Code(fmt.Sprintf("%.2f MB (%d个文件)", float64(info.TotalBytes())/(1024*1024), info.TotalResources())),
		styling.Plain("\n正在处理:\n"),
		func() styling.StyledTextOption {
			var lines []string
			for _, elem := range info.Processing() {
				lines = append(lines, fmt.Sprintf("  - %s (%.2f MB)", elem.FileName(), float64(elem.FileSize())/(1024*1024)))
			}
			if len(lines) == 0 {
				lines = append(lines, "  - 无")
			}
			return styling.Plain(slice.Join(lines, "\n"))
		}(),
		styling.Plain("\n平均速度: "),
		styling.Bold(fmt.Sprintf("%.2f MB/s", dlutil.GetSpeed(info.DownloadedBytes(), p.start)/(1024*1024))),
		styling.Plain("\n当前进度: "),
		styling.Bold(fmt.Sprintf("%.2f%%", float64(info.DownloadedBytes())/float64(info.TotalBytes())*100)),
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
			logger.Infof("Parsed task %s was canceled", info.TaskID())
			ext := tgutil.ExtFromContext(ctx)
			if ext != nil {
				ext.EditMessage(p.ChatID, &tg.MessagesEditMessageRequest{
					ID:      p.MessageID,
					Message: fmt.Sprintf("处理已取消: %s", info.TaskID()),
				})
			}
		} else {
			logger.Errorf("Parsed task %s failed: %s", info.TaskID(), err)
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
	logger.Infof("Parsed task %s completed successfully", info.TaskID())

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain("处理完成, 资源数量: "),
		styling.Code(fmt.Sprintf("%d", info.TotalResources())),
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
