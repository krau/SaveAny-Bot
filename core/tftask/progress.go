package tftask

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
)

type Progress struct {
	MessageID  int
	ChatID     int64
	client     *tg.Client
	OnStart    func(ctx context.Context, info TaskInfo)
	OnProgress func(ctx context.Context, info TaskInfo, downloaded, total int64)
	OnDone     func(ctx context.Context, info TaskInfo, err error)
}

type ProgressOption func(*Progress)

func WithProgressOnStart(fn func(ctx context.Context, info TaskInfo)) ProgressOption {
	return func(p *Progress) {
		p.OnStart = fn
	}
}

func WithProgressOnProgress(fn func(ctx context.Context, info TaskInfo, downloaded, total int64)) ProgressOption {
	return func(p *Progress) {
		p.OnProgress = fn
	}
}

func WithProgressOnDone(fn func(ctx context.Context, info TaskInfo, err error)) ProgressOption {
	return func(p *Progress) {
		p.OnDone = fn
	}
}

func NewProgressTrack(
	client *tg.Client,
	messageID int,
	chatID int64,
	opts ...ProgressOption,
) Progress {
	onStart := func(ctx context.Context, info TaskInfo) {
		log.FromContext(ctx).Debugf("Progress tracking started for message %d in chat %d", messageID, chatID)
		entityBuilder := entity.Builder{}
		var entities []tg.MessageEntityClass
		if err := styling.Perform(&entityBuilder,
			styling.Plain("开始下载\n文件名: "),
			styling.Code(info.FileName()),
			styling.Plain("\n保存路径: "),
			styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
		); err != nil {
			log.FromContext(ctx).Errorf("Failed to build entities: %s", err)
			return
		}
		text, entities := entityBuilder.Complete()
		req := &tg.MessagesEditMessageRequest{
			ID: messageID,
		}
		req.SetMessage(text)
		req.SetEntities(entities)
		ext, ok := ctx.(*ext.Context)
		if ok {
			ext.EditMessage(chatID, req)
			return
		}
	}
	onProgress := func(ctx context.Context, info TaskInfo, downloaded int64, total int64) {
		log.FromContext(ctx).Debugf("Progress update: %s, downloaded: %d, total: %d", info.FileName(), downloaded, total)
		entityBuilder := entity.Builder{}
		var entities []tg.MessageEntityClass
		if err := styling.Perform(&entityBuilder,
			styling.Plain("正在处理下载任务\n文件名: "),
			styling.Code(info.FileName()),
			styling.Plain("\n保存路径: "),
			styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
			// TODO:
			// styling.Plain("\n平均速度: "),
			// styling.Bold(getSpeed(bytesRead, info.StartTime())),
			// styling.Plain("\n当前进度: "),
			// styling.Bold(fmt.Sprintf("%.2f%%", progress)),
		); err != nil {
			log.FromContext(ctx).Errorf("Failed to build entities: %s", err)
			return
		}
		text, entities := entityBuilder.Complete()
		req := &tg.MessagesEditMessageRequest{
			ID: messageID,
		}
		req.SetMessage(text)
		req.SetEntities(entities)
		ext, ok := ctx.(*ext.Context)
		if ok {
			ext.EditMessage(chatID, req)
			return
		}
	}

	onDone := func(ctx context.Context, info TaskInfo, err error) {
		log.FromContext(ctx).Debugf("Progress done for message %d in chat %d, error: %v", messageID, chatID, err)
		if err != nil {
			entityBuilder := entity.Builder{}
			if err := styling.Perform(&entityBuilder,
				styling.Plain("下载失败\n文件名: "),
				styling.Code(info.FileName()),
				styling.Plain("\n错误: "),
				styling.Bold(err.Error()),
			); err != nil {
				log.FromContext(ctx).Errorf("Failed to build entities: %s", err)
				return
			}
			text, entities := entityBuilder.Complete()
			req := &tg.MessagesEditMessageRequest{
				ID: messageID,
			}
			req.SetMessage(text)
			req.SetEntities(entities)
			ext, ok := ctx.(*ext.Context)
			if ok {
				ext.EditMessage(chatID, req)
				return
			}
		} else {
			entityBuilder := entity.Builder{}
			if err := styling.Perform(&entityBuilder,
				styling.Plain("下载完成\n文件名: "),
				styling.Code(info.FileName()),
				styling.Plain("\n保存路径: "),
				styling.Code(fmt.Sprintf("[%s]:%s", info.StorageName(), info.StoragePath())),
			); err != nil {
				log.FromContext(ctx).Errorf("Failed to build entities: %s", err)
				return
			}
			text, entities := entityBuilder.Complete()
			req := &tg.MessagesEditMessageRequest{
				ID: messageID,
			}
			req.SetMessage(text)
			req.SetEntities(entities)
			ext, ok := ctx.(*ext.Context)
			if ok {
				ext.EditMessage(chatID, req)
				return
			}
		}
	}

	p := &Progress{
		MessageID:  messageID,
		ChatID:     chatID,
		client:     client,
		OnStart:    onStart,
		OnProgress: onProgress,
		OnDone:     onDone,
	}
	for _, opt := range opts {
		opt(p)
	}
	return *p
}

type ProgressWriterAt struct {
	ctx        context.Context
	wrAt       io.WriterAt
	progress   Progress
	downloaded *atomic.Int64
	total      int64
	info       TaskInfo
}

func (w *ProgressWriterAt) WriteAt(p []byte, off int64) (int, error) {
	at, err := w.wrAt.WriteAt(p, off)
	if err != nil {
		return 0, err
	}
	if w.progress.OnProgress == nil {
		return at, nil
	}
	w.progress.OnProgress(w.ctx, w.info, w.downloaded.Add(int64(at)), w.total)
	return at, nil
}

func newWriterAt(
	ctx context.Context,
	wrAt io.WriterAt,
	progress Progress,
	taskInfo TaskInfo,
) *ProgressWriterAt {
	return &ProgressWriterAt{
		ctx:        ctx,
		progress:   progress,
		downloaded: &atomic.Int64{},
		total:      taskInfo.FileSize(),
		wrAt:       wrAt,
		info:       taskInfo,
	}
}
