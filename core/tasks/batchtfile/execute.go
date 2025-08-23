package batchtfile

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/retry"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/common/utils/ioutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"golang.org/x/sync/errgroup"
)

func (t *Task) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("batch_file[%s]", t.ID))
	logger.Info("Starting batch file task")
	t.Progress.OnStart(ctx, t)
	workers := config.C().Workers
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(workers)
	for _, elem := range t.Elems {
		eg.Go(func() error {
			t.processingMu.RLock()
			if t.processing[elem.ID] != nil {
				return fmt.Errorf("element with ID %s is already being processed", elem.ID)
			}
			t.processingMu.RUnlock()
			t.processingMu.Lock()
			t.processing[elem.ID] = &elem
			t.processingMu.Unlock()
			defer func() {
				t.processingMu.Lock()
				delete(t.processing, elem.ID)
				t.processingMu.Unlock()
			}()
			return t.processElement(gctx, elem)
		})
	}
	err := eg.Wait()
	if err != nil {
		logger.Errorf("Error during batch file processing: %v", err)
	} else {
		logger.Info("Batch file task completed successfully")
	}
	t.Progress.OnDone(ctx, t, err)
	return err
}

func (t *Task) processElement(ctx context.Context, elem TaskElement) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", elem.File.Name()))
	if elem.stream {
		pr, pw := io.Pipe()
		defer pr.Close()
		errg, uploadCtx := errgroup.WithContext(ctx)
		errg.Go(func() error {
			return elem.Storage.Save(uploadCtx, pr, elem.Path)
		})
		wr := ioutil.NewProgressWriter(pw, func(n int) {
			t.downloaded.Add(int64(n))
			t.Progress.OnProgress(ctx, t)
		})
		errg.Go(func() error {
			defer pw.Close()
			logger.Info("Starting file download in stream mode")
			_, err := tfile.NewDownloader(elem.File).Stream(uploadCtx, wr)
			if err != nil {
				logger.Errorf("Failed to download file: %v", err)
				pw.CloseWithError(err)
			}
			return err
		})
		if err := errg.Wait(); err != nil {
			return fmt.Errorf("failed to download file in stream mode: %w", err)
		}
		logger.Info("File downloaded successfully in stream mode")
		return nil
	}
	logger.Info("Starting file download")
	localFile, err := fsutil.CreateFile(elem.localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer func() {
		if err := localFile.CloseAndRemove(); err != nil {
			logger.Errorf("Failed to close local file: %v", err)
		}
	}()
	wrAt := ioutil.NewProgressWriterAt(localFile, func(n int) {
		t.downloaded.Add(int64(n))
		t.Progress.OnProgress(ctx, t)
	})
	_, err = tfile.NewDownloader(elem.File).Parallel(ctx, wrAt)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	logger.Info("File downloaded successfully")
	if path.Ext(elem.FileName()) == "" {
		ext := fsutil.DetectFileExt(elem.localPath)
		if ext != "" {
			elem.Path = elem.Path + ext
		}
	}
	var fileStat os.FileInfo
	fileStat, err = os.Stat(elem.localPath)
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}
	vctx := context.WithValue(ctx, ctxkey.ContentLength, fileStat.Size())
	err = retry.Retry(func() error {
		var file *os.File
		file, err = os.Open(elem.localPath)
		if err != nil {
			return fmt.Errorf("failed to open cache file: %w", err)
		}
		defer file.Close()
		if err = elem.Storage.Save(vctx, file, elem.Path); err != nil {
			logger.Errorf("Failed to save file: %s, retrying...", err)
			return err
		}
		return nil
	}, retry.Context(vctx), retry.RetryTimes(uint(config.C().Retry)))
	return err
}
