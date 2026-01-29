package directlinks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sync/atomic"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/retry"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/common/utils/ioutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"golang.org/x/sync/errgroup"
)

func (t *Task) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Infof("Starting directlinks task %s", t.ID)
	if t.Progress != nil {
		t.Progress.OnStart(ctx, t)
	}
	// head all links to get file info
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(config.C().Workers)
	fetchedTotalBytes := atomic.Int64{}
	for _, file := range t.files {
		eg.Go(func() error {
			req, err := http.NewRequestWithContext(ctx, http.MethodHead, file.URL, nil)
			if err != nil {
				return fmt.Errorf("failed to create HEAD request for %s: %w", file.URL, err)
			}
			resp, err := t.client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to HEAD %s: %w", file.URL, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("HEAD %s returned status %d", file.URL, resp.StatusCode)
			}
			fetchedTotalBytes.Add(resp.ContentLength)
			file.Size = resp.ContentLength
			if name := resp.Header.Get("Content-Disposition"); name != "" {
				filename := parseFilename(name)
				if filename != "" {
					file.Name = filename
				}
			}
			// extract filename from URL if Content-Disposition is empty or invalid
			if file.Name == "" {
				file.Name = parseFilenameFromURL(file.URL)
			}
			if file.Name == "" {
				return fmt.Errorf("failed to determine filename for %s: Content-Disposition header is empty and URL does not contain a valid filename", file.URL)
			}

			return nil
		})
	}
	err := eg.Wait()
	if err != nil {
		logger.Errorf("Error during HEAD requests: %v", err)
		if t.Progress != nil {
			t.Progress.OnDone(ctx, t, err)
		}
		return err
	}
	t.totalBytes = fetchedTotalBytes.Load()
	// start downloading
	eg, gctx = errgroup.WithContext(ctx)
	eg.SetLimit(config.C().Workers)
	for _, file := range t.files {
		eg.Go(func() error {
			t.processingMu.RLock()
			if _, ok := t.processing[file.URL]; ok {
				return fmt.Errorf("file %s is already being processed", file.URL)
			}
			t.processingMu.RUnlock()
			t.processingMu.Lock()
			t.processing[file.URL] = file
			t.processingMu.Unlock()
			defer func() {
				t.processingMu.Lock()
				delete(t.processing, file.URL)
				t.processingMu.Unlock()
			}()
			err := t.processLink(gctx, file)
			t.downloaded.Add(1)
			if errors.Is(err, context.Canceled) {
				logger.Debug("Link processing canceled")
				return err
			}
			if err != nil {
				logger.Errorf("Error processing link %s: %v", file.URL, err)
				return fmt.Errorf("failed to process link %s: %w", file.URL, err)
			}
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		logger.Errorf("Error during directlinks task execution: %v", err)
	} else {
		logger.Infof("Directlinks task %s completed successfully", t.ID)
	}
	if t.Progress != nil {
		t.Progress.OnDone(ctx, t, err)
	}
	return err
}

func (t *Task) processLink(ctx context.Context, file *File) error {
	logger := log.FromContext(ctx)
	err := retry.Retry(func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.URL, nil)
		if err != nil {
			return fmt.Errorf("failed to create GET request for %s: %w", file.URL, err)
		}
		resp, err := t.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to GET %s: %w", file.URL, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("GET %s returned status %d", file.URL, resp.StatusCode)
		}
		ctx = context.WithValue(ctx, ctxkey.ContentLength, file.Size)
		if t.stream {
			return t.Storage.Save(ctx, resp.Body, filepath.Join(t.StorPath, file.Name))
		}
		cacheFile, err := fsutil.CreateFile(filepath.Join(config.C().Temp.BasePath,
			fmt.Sprintf("direct_%s_%s", t.ID, file.Name)))
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer func() {
			if err := cacheFile.CloseAndRemove(); err != nil {
				logger.Errorf("Failed to close and remove cache file: %v", err)
			}
		}()
		wr := ioutil.NewProgressWriter(cacheFile, func(n int) {
			t.downloadedBytes.Add(int64(n))
			if t.Progress != nil {
				t.Progress.OnProgress(ctx, t)
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
				return fmt.Errorf("failed to copy file %s to cache file: %w", file.URL, err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		_, err = cacheFile.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("failed to seek cache file for resource %s: %w", file.URL, err)
		}
		return t.Storage.Save(ctx, cacheFile, filepath.Join(t.StorPath, file.Name))
	}, retry.RetryTimes(uint(config.C().Retry)), retry.Context(ctx))
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
