package tphtask

import (
	"context"
	"fmt"
	"path"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/retry"
	"github.com/krau/SaveAny-Bot/config"
	"golang.org/x/sync/errgroup"
)

func (t *Task) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Infof("Starting Telegraph task %s", t.PhPath)
	t.progress.OnStart(ctx, t)
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(config.Cfg.Workers)
	for i, pic := range t.Pics {
		pic := pic
		i := i
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
		retry.RetryTimes(uint(config.Cfg.Retry)),
	}
	err := retry.Retry(func() error {
		// main logic
		body, err := t.client.Download(ctx, picUrl)
		if err != nil {
			return fmt.Errorf("failed to download picture %s: %w", picUrl, err)
		}
		defer body.Close()
		filename := fmt.Sprintf("%d%s", index+1, path.Ext(picUrl))
		return t.Stor.Save(ctx, body, path.Join(t.StorPath, filename))
	}, retryOpts...)
	return err
}
