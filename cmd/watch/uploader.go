package watch

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/storage"
)

type uploadJob struct {
	// localPath is the absolute path of the local file.
	localPath string
	// relPath is relative to the watch root, used to preserve directory structure on storage.
	relPath string
}

// Uploader uploads local files to the target storage via a worker pool.
// If a file changes while being uploaded, it is re-uploaded once after the
// current upload finishes, instead of being queued multiple times.
type Uploader struct {
	stor       storage.Storage
	destDir    string
	overwrite  bool
	retry      int
	retryDelay time.Duration
	logger     *log.Logger

	jobs chan uploadJob
	wg   sync.WaitGroup

	mu sync.Mutex
	// inflight maps in-progress (or queued) file paths. A true value means the
	// file changed during upload and must be re-queued once done.
	inflight map[string]bool
}

type UploaderOptions struct {
	Storage    storage.Storage
	DestDir    string
	Overwrite  bool
	Workers    int
	Retry      int
	RetryDelay time.Duration
	QueueSize  int
}

// NewUploader creates and starts an Uploader. The caller must call Close when done.
func NewUploader(ctx context.Context, opts UploaderOptions) *Uploader {
	if opts.Workers < 1 {
		opts.Workers = 1
	}
	if opts.Retry < 1 {
		opts.Retry = 1
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = 3 * time.Second
	}
	if opts.QueueSize < opts.Workers {
		opts.QueueSize = opts.Workers * 64
	}

	u := &Uploader{
		stor:       opts.Storage,
		destDir:    opts.DestDir,
		overwrite:  opts.Overwrite,
		retry:      opts.Retry,
		retryDelay: opts.RetryDelay,
		logger:     log.FromContext(ctx).WithPrefix("uploader"),
		jobs:       make(chan uploadJob, opts.QueueSize),
		inflight:   make(map[string]bool),
	}

	for i := 0; i < opts.Workers; i++ {
		u.wg.Add(1)
		go u.worker(ctx)
	}

	return u
}

// Submit enqueues an upload job. If the same file is already in flight, it is
// marked for re-upload instead of being queued again. Returns false if ctx is
// cancelled before the job can be enqueued.
func (u *Uploader) Submit(ctx context.Context, job uploadJob) bool {
	u.mu.Lock()
	if _, ok := u.inflight[job.localPath]; ok {
		u.inflight[job.localPath] = true
		u.mu.Unlock()
		u.logger.Debugf("file %s already in flight, marked for re-upload", job.localPath)
		return true
	}
	u.inflight[job.localPath] = false
	u.mu.Unlock()

	select {
	case u.jobs <- job:
		return true
	case <-ctx.Done():
		u.mu.Lock()
		delete(u.inflight, job.localPath)
		u.mu.Unlock()
		return false
	}
}

func (u *Uploader) worker(ctx context.Context) {
	defer u.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-u.jobs:
			if !ok {
				return
			}
			u.process(ctx, job)
		}
	}
}

func (u *Uploader) process(ctx context.Context, job uploadJob) {
	if err := u.uploadWithRetry(ctx, job); err != nil {
		if ctx.Err() != nil {
			u.clearInflight(job.localPath)
			return
		}
		u.logger.Errorf("failed to upload %s after %d attempt(s): %v", job.localPath, u.retry, err)
	}

	// Re-queue if the file changed again while it was being uploaded.
	u.mu.Lock()
	needReupload := u.inflight[job.localPath]
	if needReupload {
		u.inflight[job.localPath] = false
	} else {
		delete(u.inflight, job.localPath)
	}
	u.mu.Unlock()

	if needReupload {
		select {
		case u.jobs <- job:
			u.logger.Debugf("re-queued %s due to changes during upload", job.localPath)
		case <-ctx.Done():
			u.clearInflight(job.localPath)
		}
	}
}

func (u *Uploader) clearInflight(localPath string) {
	u.mu.Lock()
	delete(u.inflight, localPath)
	u.mu.Unlock()
}

func (u *Uploader) uploadWithRetry(ctx context.Context, job uploadJob) error {
	var lastErr error
	for attempt := 1; attempt <= u.retry; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := u.upload(ctx, job)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		lastErr = err
		u.logger.Warnf("upload %s failed (attempt %d/%d): %v", job.localPath, attempt, u.retry, err)
		if attempt < u.retry {
			select {
			case <-time.After(u.retryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return lastErr
}

func (u *Uploader) upload(ctx context.Context, job uploadJob) error {
	file, err := os.Open(filepath.Clean(job.localPath))
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	// Keep the relative directory structure on the storage side.
	storagePath := path.Join(u.destDir, filepath.ToSlash(job.relPath))

	uploadCtx := context.WithValue(ctx, ctxkey.ContentLength, info.Size())
	if u.overwrite {
		uploadCtx = storage.WithOverwrite(uploadCtx)
	} else if u.stor.Exists(uploadCtx, storagePath) {
		u.logger.Infof("skip existing file: %s", storagePath)
		return nil
	}

	u.logger.Infof("uploading %s -> %s (%d bytes)", job.localPath, storagePath, info.Size())
	if err := u.stor.Save(uploadCtx, file, storagePath); err != nil {
		return fmt.Errorf("failed to save to storage: %w", err)
	}
	u.logger.Infof("uploaded %s", storagePath)
	return nil
}

// Close stops accepting jobs and waits for in-flight uploads to finish.
func (u *Uploader) Close() {
	close(u.jobs)
	u.wg.Wait()
}
