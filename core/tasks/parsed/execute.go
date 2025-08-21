package parsed

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/parser"
	"golang.org/x/sync/errgroup"
)

func (t *Task) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Infof("Starting Parsed item task %s", t.item.Title)
	// t.progress.OnStart(ctx, t)
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(config.Cfg.Workers)
	for _, resource := range t.item.Resources {
		eg.Go(func() error {
			err := t.processResource(gctx, resource)
			if err != nil {
				logger.Errorf("Error processing resource %s: %v", resource.URL, err)
				return fmt.Errorf("failed to process resource %s: %w", resource.URL, err)
			}
			t.downloaded.Add(1)
			// t.progress.OnProgress(gctx, t)
			return nil
		})
	}
	err := eg.Wait()
	if err != nil {
		logger.Errorf("Error during Parsed item task execution: %v", err)
	} else {
		logger.Infof("Parsed item task %s completed successfully", t.item.Title)
	}
	// t.progress.OnDone(ctx, t, err)
	return err
}

func (t *Task) processResource(ctx context.Context, resource parser.Resource) error {
	logger := log.FromContext(ctx)
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
	cacheFile, err := fsutil.CreateFile(filepath.Join(config.Cfg.Temp.BasePath,
		fmt.Sprintf("resource_%s_%s", t.ID, resource.Filename)))
	if err != nil {
		return fmt.Errorf("failed to create cache file for resource %s: %w", resource.URL, err)
	}
	defer func() {
		if err := cacheFile.CloseAndRemove(); err != nil {
			logger.Errorf("Failed to close and remove cache file: %v", err)
		}
	}()
	_, err = io.Copy(cacheFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy resource %s to cache file: %w", resource.URL, err)
	}
	_, err = cacheFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek cache file for resource %s: %w", resource.URL, err)
	}
	return t.Stor.Save(ctx, cacheFile, path.Join(t.StorPath, resource.Filename))
}
