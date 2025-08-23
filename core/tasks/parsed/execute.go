package parsed

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/retry"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/common/utils/ioutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/pkg/parser"
	"golang.org/x/sync/errgroup"
)

func (t *Task) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Infof("Starting Parsed item task %s", t.item.Title)
	if t.progress != nil {
		t.progress.OnStart(ctx, t)
	}
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(config.C().Workers)
	for _, resource := range t.item.Resources {
		eg.Go(func() error {
			t.processingMu.RLock()
			if t.processing[resource.ID()] != nil {
				return fmt.Errorf("resource %s is already being processed", resource.ID())
			}
			t.processingMu.RUnlock()
			t.processingMu.Lock()
			t.processing[resource.ID()] = &resource
			t.processingMu.Unlock()
			defer func() {
				t.processingMu.Lock()
				delete(t.processing, resource.URL)
				t.processingMu.Unlock()
			}()
			err := t.processResource(gctx, resource)
			t.downloaded.Add(1)
			if errors.Is(err, context.Canceled) {
				logger.Debug("Resource processing canceled")
				return err
			}
			if err != nil {
				logger.Errorf("Error processing resource %s: %v", resource.URL, err)
				return fmt.Errorf("failed to process resource %s: %w", resource.URL, err)
			}
			return nil
		})
	}
	err := eg.Wait()
	if err != nil {
		logger.Errorf("Error during Parsed item task execution: %v", err)
	} else {
		logger.Infof("Parsed item task %s completed successfully", t.item.Title)
	}
	if t.progress != nil {
		t.progress.OnDone(ctx, t, err)
	}
	return err
}

func (t *Task) processResource(ctx context.Context, resource parser.Resource) error {
	logger := log.FromContext(ctx)
	err := retry.Retry(func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, resource.URL, nil)
		if err != nil {
			return err
		}
		if resource.Headers != nil {
			for k, v := range resource.Headers {
				req.Header.Set(k, v)
			}
		}
		resp, err := t.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to download resource %s: %w", resource.URL, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download resource %s: %s", resource.URL, resp.Status)
		}
		ctx = context.WithValue(ctx, ctxkey.ContentLength, func() int64 {
			if resource.Size > 0 {
				return resource.Size
			}
			return resp.ContentLength
		}())
		if t.stream {
			return t.Stor.Save(ctx, resp.Body, path.Join(t.StorPath, resource.Filename))
		}
		cacheFile, err := fsutil.CreateFile(filepath.Join(config.C().Temp.BasePath,
			fmt.Sprintf("resource_%s_%s", t.ID, resource.Filename)))
		if err != nil {
			return fmt.Errorf("failed to create cache file for resource %s: %w", resource.URL, err)
		}
		defer func() {
			if err := cacheFile.CloseAndRemove(); err != nil {
				logger.Errorf("Failed to close and remove cache file: %v", err)
			}
		}()
		wr := ioutil.NewProgressWriter(cacheFile, func(n int) {
			t.downloadedBytes.Add(int64(n))
			if t.progress != nil {
				t.progress.OnProgress(ctx, t)
			}
		})

		copyResultCh := make(chan error, 1)
		go func() {
			_, err := io.Copy(wr, resp.Body)
			copyResultCh <- err
		}()
		select {
		case err := <-copyResultCh:
			if err != nil {
				return fmt.Errorf("failed to copy resource %s to cache file: %w", resource.URL, err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		_, err = cacheFile.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("failed to seek cache file for resource %s: %w", resource.URL, err)
		}
		return t.Stor.Save(ctx, cacheFile, path.Join(t.StorPath, resource.Filename))
	}, retry.Context(ctx), retry.RetryTimes(uint(config.C().Retry)))
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
