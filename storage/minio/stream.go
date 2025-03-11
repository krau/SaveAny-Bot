package minio

import (
	"context"
	"fmt"
	"io"

	"github.com/krau/SaveAny-Bot/logger"
	"github.com/minio/minio-go/v7"
)

type MinioWriter struct {
	pipeWriter *io.PipeWriter
	done       chan error
	path       string
	ctx        context.Context
	closed     bool
}

func (w *MinioWriter) Write(p []byte) (n int, err error) {
	select {
	case <-w.ctx.Done():
		return 0, w.ctx.Err()
	default:
		return w.pipeWriter.Write(p)
	}
}

func (w *MinioWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	if err := w.pipeWriter.Close(); err != nil {
		return fmt.Errorf("failed to close pipe writer: %w", err)
	}

	select {
	case err := <-w.done:
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}
		return nil
	case <-w.ctx.Done():
		return fmt.Errorf("upload cancelled: %w", w.ctx.Err())
	}
}

func (m *Minio) NewUploadStream(ctx context.Context, storagePath string) (io.WriteCloser, error) {
	logger.L.Infof("Creating upload stream for %s", storagePath)

	uploadCtx, cancel := context.WithCancel(ctx)
	pipeReader, pipeWriter := io.Pipe()
	done := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic during upload: %v", r)
			}
			pipeReader.Close()
			cancel()
		}()

		info, err := m.client.PutObject(
			uploadCtx,
			m.config.BucketName,
			storagePath,
			pipeReader,
			-1,
			minio.PutObjectOptions{},
		)

		if err != nil {
			logger.L.Errorf("Failed to upload to %s: %v", storagePath, err)
			done <- err
			return
		}

		logger.L.Infof("uploaded %d bytes to %s", info.Size, storagePath)
		done <- nil
	}()

	return &MinioWriter{
		pipeWriter: pipeWriter,
		done:       done,
		path:       storagePath,
		ctx:        uploadCtx,
		closed:     false,
	}, nil
}
