package telegram

import (
	"context"
	"sync"

	"github.com/gotd/td/telegram/uploader"
)

var _ uploader.Progress = (*uploadProgress)(nil)

type uploadProgress struct {
	mu         sync.Mutex
	onProgress func(uploaded, total int64)
	total      int64
	uploaded   int64
	byID       map[int64]int64
}

func newUploadProgress(total int64, onProgress func(uploaded, total int64)) *uploadProgress {
	return &uploadProgress{
		onProgress: onProgress,
		total:      total,
		byID:       make(map[int64]int64),
	}
}

func (p *uploadProgress) Chunk(ctx context.Context, state uploader.ProgressState) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	p.mu.Lock()
	previous := p.byID[state.ID]
	if state.Uploaded <= previous {
		p.mu.Unlock()
		return nil
	}
	p.byID[state.ID] = state.Uploaded
	p.uploaded += state.Uploaded - previous
	uploaded := p.uploaded
	total := p.total
	if total <= 0 {
		total = state.Total
	}
	p.mu.Unlock()

	if p.onProgress != nil {
		p.onProgress(uploaded, total)
	}
	return nil
}

func (p *uploadProgress) reset(total int64) {
	p.mu.Lock()
	p.total = total
	p.uploaded = 0
	p.byID = make(map[int64]int64)
	p.mu.Unlock()
}
