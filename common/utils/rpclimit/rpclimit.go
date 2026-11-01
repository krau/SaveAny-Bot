// Package rpclimit bounds the number of concurrent Telegram file-part RPCs.
//
// Per-file threads alone do not bound concurrency: every task worker can run
// several elements, and each element opens its own set of part requests, so
// the real load is workers × elements × threads. Downloads and uploads share
// one budget derived from the configuration, which is re-read on every
// acquire so runtime setting changes apply without a restart.
package rpclimit

import (
	"context"
	"sync"

	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/config"
)

var transfer = &limiter{notify: make(chan struct{})}

// Acquire reserves one transfer slot. It blocks until a slot is free or ctx is
// done. Every successful Acquire must be paired with Release.
func Acquire(ctx context.Context) error {
	return transfer.acquire(ctx, config.TransferBudget())
}

// Release returns a previously acquired slot.
func Release() {
	transfer.release()
}

// WrapDownloader limits UploadGetFile calls made through the returned client.
func WrapDownloader(client downloader.Client) downloader.Client {
	if client == nil {
		return nil
	}
	return limitedDownloader{Client: client}
}

// WrapUploader limits save-part calls made through the returned uploader
// client.
func WrapUploader(client uploader.Client) uploader.Client {
	if client == nil {
		return nil
	}
	return limitedUploader{Client: client}
}

type limitedDownloader struct {
	downloader.Client
}

func (c limitedDownloader) UploadGetFile(ctx context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	if err := Acquire(ctx); err != nil {
		return nil, err
	}
	defer Release()
	return c.Client.UploadGetFile(ctx, req)
}

type limitedUploader struct {
	uploader.Client
}

func (c limitedUploader) UploadSaveFilePart(ctx context.Context, req *tg.UploadSaveFilePartRequest) (bool, error) {
	if err := Acquire(ctx); err != nil {
		return false, err
	}
	defer Release()
	return c.Client.UploadSaveFilePart(ctx, req)
}

func (c limitedUploader) UploadSaveBigFilePart(ctx context.Context, req *tg.UploadSaveBigFilePartRequest) (bool, error) {
	if err := Acquire(ctx); err != nil {
		return false, err
	}
	defer Release()
	return c.Client.UploadSaveBigFilePart(ctx, req)
}

// limiter is a context-aware counting semaphore with a dynamic limit.
type limiter struct {
	mu       sync.Mutex
	inFlight int
	notify   chan struct{}
}

func (l *limiter) acquire(ctx context.Context, limit int) error {
	limit = max(limit, 1)
	for {
		l.mu.Lock()
		if l.inFlight < limit {
			l.inFlight++
			l.mu.Unlock()
			return nil
		}
		wait := l.notify
		l.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wait:
		}
	}
}

func (l *limiter) release() {
	l.mu.Lock()
	if l.inFlight > 0 {
		l.inFlight--
	}
	l.broadcastLocked()
	l.mu.Unlock()
}

func (l *limiter) broadcastLocked() {
	close(l.notify)
	l.notify = make(chan struct{})
}
