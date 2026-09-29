package tgutil

import (
	"context"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"golang.org/x/time/rate"
)

// ProgressSender performs one progress message edit. It may block for minutes
// when Telegram asks the client to wait (FLOOD_WAIT).
type ProgressSender func(ctx context.Context, chatID int64, req *tg.MessagesEditMessageRequest) error

const progressEditInterval = time.Second

var progressEditLimiter = rate.NewLimiter(rate.Every(progressEditInterval), 1)

func SendProgressEdit(ctx context.Context, chatID int64, req *tg.MessagesEditMessageRequest) error {
	if err := progressEditLimiter.Wait(context.Background()); err != nil {
		return err
	}
	ext := ExtFromContext(ctx)
	if ext == nil {
		return nil
	}
	if _, err := ext.EditMessage(chatID, req); err != nil {
		return err
	}
	return nil
}

type progressEditJob struct {
	ctx    context.Context
	chatID int64
	req    *tg.MessagesEditMessageRequest
	final  bool
}

type ProgressEditQueue struct {
	sender ProgressSender
	slot   chan progressEditJob
	start  sync.Once
}

func NewProgressEditQueue(sender ProgressSender) *ProgressEditQueue {
	return &ProgressEditQueue{
		sender: sender,
		slot:   make(chan progressEditJob, 1),
	}
}

func (q *ProgressEditQueue) Submit(ctx context.Context, chatID int64, req *tg.MessagesEditMessageRequest, final bool) {
	job := progressEditJob{ctx: ctx, chatID: chatID, req: req, final: final}
	q.start.Do(func() { go q.run() })
	for {
		select {
		case q.slot <- job:
			return
		default:
		}
		select {
		case pending := <-q.slot:
			if pending.final {
				// The final edit must stay last; nothing may be queued after it.
				q.slot <- pending
				return
			}
		default:
		}
	}
}

func (q *ProgressEditQueue) run() {
	for job := range q.slot {
		q.perform(job)
		if job.final {
			return
		}
	}
}

func (q *ProgressEditQueue) perform(job progressEditJob) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.FromContext(job.ctx).Errorf("Recovered from panic while editing progress message: %v", recovered)
		}
	}()
	if err := q.sender(job.ctx, job.chatID, job.req); err != nil {
		log.FromContext(job.ctx).Errorf("Failed to edit progress message: %v", err)
	}
}
