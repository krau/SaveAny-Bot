package transfer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/storage"
	"golang.org/x/sync/errgroup"
)

// Execute implements core.Executable.
func (t *Task) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("transfer[%s]", t.ID))
	logger.Info("Starting transfer task")
	t.Progress.OnStart(ctx, t)

	workers := config.C().Workers
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(workers)

	for _, elem := range t.elems {
		eg.Go(func() error {
			t.processingMu.RLock()
			if t.processing[elem.ID] != nil {
				t.processingMu.RUnlock()
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

			err := t.processElement(gctx, elem)
			if err != nil && !t.IgnoreErrors {
				return err
			}
			if err != nil {
				t.processingMu.Lock()
				t.failed[elem.ID] = err
				t.processingMu.Unlock()
				logger.Errorf("Failed to process file %s: %v", elem.FileInfo.Name, err)
			}
			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		logger.Errorf("Error during transfer processing: %v", err)
	} else {
		logger.Info("Transfer task completed successfully")
	}

	t.Progress.OnDone(ctx, t, err)
	return err
}

func (t *Task) processElement(ctx context.Context, elem TaskElement) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", elem.FileInfo.Name))

	// Check whether the source storage supports reading
	readableStorage, ok := elem.SourceStorage.(storage.StorageReadable)
	if !ok {
		return fmt.Errorf("source storage %s does not support reading", elem.SourceStorage.Name())
	}

	logger.Info("Opening file from source storage")
	reader, size, err := readableStorage.OpenFile(ctx, elem.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	// Build target storage path: /target_path/filename
	storagePath := path.Join(elem.TargetPath, elem.FileInfo.Name)

	// Inject file size into context
	ctx = context.WithValue(ctx, ctxkey.ContentLength, size)

	if config.C().Stream {
		if err := elem.TargetStorage.Save(ctx, reader, storagePath); err != nil {
			return fmt.Errorf("failed to upload file to storage: %w", err)
		}
	} else {
		logger.Info("Downloading to temporary file for ReadSeeker support")
		tempFile, err := t.downloadToTemp(reader, elem.FileInfo.Name)
		if err != nil {
			return fmt.Errorf("failed to download to temp: %w", err)
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek temp file: %w", err)
		}

		logger.Infof("Uploading file to storage (size: %d bytes)", size)
		if err := elem.TargetStorage.Save(ctx, tempFile, storagePath); err != nil {
			return fmt.Errorf("failed to upload file to storage: %w", err)
		}
	}

	t.uploaded.Add(size)
	t.Progress.OnProgress(ctx, t)

	logger.Info("File uploaded successfully")
	return nil
}

func (t *Task) downloadToTemp(reader io.Reader, filename string) (*os.File, error) {
	tempDir := config.C().Temp.BasePath
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	tempFile, err := os.CreateTemp(tempDir, filepath.Base(filename)+"-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := io.Copy(tempFile, reader); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("failed to copy to temp file: %w", err)
	}

	return tempFile, nil
}
