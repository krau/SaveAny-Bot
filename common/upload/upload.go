// Package upload provides the shared local-file upload path used by the CLI
// commands. It keeps retries, cancellation, and confirmed progress consistent
// across one-shot and watch uploads.
package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/krau/SaveAny-Bot/common/utils/ioutil"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/storage"
)

type Options struct {
	Storage    storage.Storage
	Overwrite  bool
	Retries    int
	RetryDelay time.Duration
	OnProgress func(uploaded, total int64)
}

func (o Options) validate() error {
	if o.Storage == nil {
		return errors.New("upload: storage is required")
	}
	if o.Retries < 1 {
		return errors.New("upload: retries must be at least 1")
	}
	if o.RetryDelay < 0 {
		return errors.New("upload: retry delay must not be negative")
	}
	return nil
}

// SaveFile uploads a local file, rewinding it for every retry.
func SaveFile(ctx context.Context, localPath, storagePath string, opts Options) (retErr error) {
	if err := opts.validate(); err != nil {
		return err
	}
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("upload: open %s: %w", localPath, err)
	}
	defer func() {
		retErr = errors.Join(retErr, file.Close())
	}()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("upload: stat %s: %w", localPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("upload: %s is a directory", localPath)
	}
	return Save(ctx, file, info.Size(), storagePath, opts)
}

// Save uploads a seekable reader. The reader is rewound before each attempt so
// a retry never sends a partially consumed stream.
func Save(ctx context.Context, reader io.ReadSeeker, size int64, storagePath string, opts Options) error {
	if err := opts.validate(); err != nil {
		return err
	}
	uploadCtx := context.WithValue(ctx, ctxkey.ContentLength, size)
	if opts.Overwrite {
		uploadCtx = storage.WithOverwrite(uploadCtx)
	}

	var lastErr error
	for attempt := 0; attempt < opts.Retries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("upload: rewind: %w", err)
		}
		err := saveOnce(uploadCtx, reader, size, storagePath, opts)
		if err == nil {
			return nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		lastErr = err
		if attempt+1 >= opts.Retries {
			break
		}
		if opts.RetryDelay > 0 {
			timer := time.NewTimer(opts.RetryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return fmt.Errorf("upload failed after %d attempt(s): %w", opts.Retries, lastErr)
}

func saveOnce(ctx context.Context, reader io.ReadSeeker, size int64, storagePath string, opts Options) error {
	if progressSaver, ok := opts.Storage.(storage.StorageProgressSaver); ok {
		return progressSaver.SaveWithProgress(ctx, reader, storagePath, opts.OnProgress)
	}
	if opts.OnProgress == nil {
		return opts.Storage.Save(ctx, reader, storagePath)
	}
	progressReader := ioutil.NewProgressReader(reader, size, opts.OnProgress)
	return opts.Storage.Save(ctx, progressReader, storagePath)
}
