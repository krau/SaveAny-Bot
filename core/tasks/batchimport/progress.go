package batchimport

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
	log.FromContext(ctx).Debugf("Batch import task progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)

	entityBuilder := entity.Builder{}
	if err := styling.Perform(&entityBuilder,
		styling.Plain("正在导入: "),
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
		},
	})

	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		ext.EditMessage(p.ChatID, req)
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

	progressText.WriteString(fmt.Sprintf("导入进度: %d%%\n", percent))
	progressText.WriteString(fmt.Sprintf("已上传: %.2f MB / %.2f MB\n",
		float64(info.Uploaded())/(1024*1024),
		float64(info.TotalSize())/(1024*1024)))

	if p.start.Unix() > 0 {
		elapsed := time.Since(p.start)
		speed := float64(info.Uploaded()) / elapsed.Seconds()
		progressText.WriteString(fmt.Sprintf("速度: %s/s\n", dlutil.FormatSize(int64(speed))))

		if info.Uploaded() > 0 {
			remaining := time.Duration(float64(info.TotalSize()-info.Uploaded()) / speed * float64(time.Second))
			progressText.WriteString(fmt.Sprintf("剩余时间: %s\n", formatDuration(remaining)))
		}
	}

	processing := info.Processing()
	if len(processing) > 0 {
		progressText.WriteString("\n正在处理:\n")
		for i, elem := range processing {
			if i >= 3 {
				progressText.WriteString(fmt.Sprintf("...和其他 %d 个文件\n", len(processing)-3))
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
	log.FromContext(ctx).Debugf("Batch import task progress tracking done for message %d in chat %d", p.MessageID, p.ChatID)

	entityBuilder := entity.Builder{}
	var resultText strings.Builder

	if err != nil {
		resultText.WriteString("❌ 导入失败\n")
		fmt.Fprintf(&resultText, "错误: %v\n", err)
	} else {
		resultText.WriteString("✅ 导入完成\n")
	}

	elapsed := time.Since(p.start)
	resultText.WriteString(fmt.Sprintf("\n总文件数: %d\n", info.Count()))
	resultText.WriteString(fmt.Sprintf("总大小: %.2f MB\n", float64(info.TotalSize())/(1024*1024)))
	resultText.WriteString(fmt.Sprintf("已上传: %.2f MB\n", float64(info.Uploaded())/(1024*1024)))
	resultText.WriteString(fmt.Sprintf("耗时: %s\n", formatDuration(elapsed)))

	if elapsed.Seconds() > 0 {
		avgSpeed := float64(info.Uploaded()) / elapsed.Seconds()
		resultText.WriteString(fmt.Sprintf("平均速度: %s/s\n", dlutil.FormatSize(int64(avgSpeed))))
	}

	failedFiles := info.FailedFiles()
	if len(failedFiles) > 0 {
		fmt.Fprintf(&resultText, "\n失败文件数: %d\n", len(failedFiles))
		for i, name := range failedFiles {
			if i >= 5 {
				fmt.Fprintf(&resultText, "...和其他 %d 个文件\n", len(failedFiles)-5)
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
