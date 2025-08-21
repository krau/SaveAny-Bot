package tfile

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"golang.org/x/sync/errgroup"
)

func executeStream(ctx context.Context, task *Task) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", task.File.Name()))

	pr, pw := io.Pipe()
	defer pr.Close()
	errg, uploadCtx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		return task.Storage.Save(uploadCtx, pr, task.Path)
	})
	wr := newWriter(ctx, pw, task.Progress, task)
	errg.Go(func() error {
		defer pw.Close()
		logger.Info("Starting file download in stream mode")
		_, err := tfile.NewDownloader(task.File).Stream(uploadCtx, wr)
		if err != nil {
			logger.Errorf("Failed to download file: %v", err)
			pw.CloseWithError(err)
		}
		return err
	})
	var err error
	defer func() {
		if task.Progress != nil {
			task.Progress.OnDone(ctx, task, err)
		}
	}()
	if err = errg.Wait(); err != nil {
		return err
	}
	logger.Info("File downloaded successfully in stream mode")
	return nil
}
