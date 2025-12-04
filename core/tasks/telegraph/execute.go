package telegraph

import (
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/retry"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/config"
	"golang.org/x/sync/errgroup"
)

func (t *Task) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Infof("Starting Telegraph task %s", t.PhPath)
	t.progress.OnStart(ctx, t)
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(config.C().Workers)
	for i, pic := range t.Pics {
		eg.Go(func() error {
			err := t.processPic(gctx, pic, i)
			if err != nil {
				logger.Errorf("Error processing picture %s: %v", pic, err)
				return fmt.Errorf("failed to process picture %s: %w", pic, err)
			}
			t.downloaded.Add(1)
			t.progress.OnProgress(gctx, t)
			return nil
		})
	}
	err := eg.Wait()
	if err != nil {
		logger.Errorf("Error during Telegraph task execution: %v", err)
	} else {
		logger.Infof("Telegraph task %s completed successfully", t.PhPath)
	}
	t.progress.OnDone(ctx, t, err)
	return err
}

func (t *Task) processPic(ctx context.Context, picUrl string, index int) error {
	retryOpts := []retry.Option{
		retry.Context(ctx),
		retry.RetryTimes(uint(config.C().Retry)),
	}
	err := retry.Retry(func() error {
		body, err := t.client.Download(ctx, picUrl)
		if err != nil {
			return fmt.Errorf("failed to download picture %s: %w", picUrl, err)
		}
		defer body.Close()
		filename := fmt.Sprintf("%d%s", index+1, path.Ext(picUrl))
		if t.cannotStream {
			cacheFile, err := fsutil.CreateFile(filepath.Join(config.C().Temp.BasePath,
				fmt.Sprintf("tph_%s_%s", t.TaskID(), filename),
			))
			if err != nil {
				return fmt.Errorf("failed to create cache file for picture %s: %w", filename, err)
			}
			defer func() {
				if err := cacheFile.CloseAndRemove(); err != nil {
					logger := log.FromContext(ctx)
					logger.Errorf("Failed to close and remove cache file for picture %s: %v", filename, err)
				}
			}()
			_, err = io.Copy(cacheFile, body)
			if err != nil {
				return fmt.Errorf("failed to copy picture %s to cache file: %w", filename, err)
			}
			_, err = cacheFile.Seek(0, 0)
			if err != nil {
				return fmt.Errorf("failed to seek cache file for picture %s: %w", filename, err)
			}
			err = t.Stor.Save(ctx, cacheFile, path.Join(t.StorPath, filename))
			if err != nil {
				return fmt.Errorf("failed to save picture %s: %w", filename, err)
			}
		} else {
			err = t.Stor.Save(ctx, body, path.Join(t.StorPath, filename))
		}

		if err != nil {
			return fmt.Errorf("failed to save picture %s: %w", filename, err)
		}
		return nil
	}, retryOpts...)
	return err
}
