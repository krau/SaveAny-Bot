package aria2dl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/aria2"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
)

// Execute implements core.Executable.
func (t *Task) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Infof("Starting aria2 download task %s (GID: %s)", t.ID, t.GID)

	if t.Progress != nil {
		t.Progress.OnStart(ctx, t)
	}

	// Wait for aria2 download to complete
	if err := t.waitForDownload(ctx); err != nil {
		// If context was canceled, also cancel the aria2 download
		if errors.Is(err, context.Canceled) {
			t.cancelAria2Download()
		}
		logger.Errorf("Aria2 download failed: %v", err)
		if t.Progress != nil {
			t.Progress.OnDone(ctx, t, err)
		}
		return err
	}

	// Transfer downloaded files to storage
	if err := t.transferFiles(ctx); err != nil {
		logger.Errorf("File transfer failed: %v", err)
		if t.Progress != nil {
			t.Progress.OnDone(ctx, t, err)
		}
		return err
	}

	logger.Infof("Aria2 task %s completed successfully", t.ID)
	if t.Progress != nil {
		t.Progress.OnDone(ctx, t, nil)
	}

	// Clean up aria2 download result
	if _, err := t.Aria2Client.RemoveDownloadResult(context.Background(), t.GID); err != nil {
		logger.Warnf("Failed to remove aria2 download result: %v", err)
	}

	return nil
}

// waitForDownload waits for aria2 to complete the download
func (t *Task) waitForDownload(ctx context.Context) error {
	logger := log.FromContext(ctx)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := t.getStatus(ctx)
			if err != nil {
				return err
			}

			if t.Progress != nil {
				t.Progress.OnProgress(ctx, t, status)
			}

			// Check if download is complete
			if status.IsDownloadComplete() {
				// Handle metadata downloads (torrent/magnet) that spawn follow-up downloads
				if len(status.FollowedBy) > 0 {
					logger.Infof("Switching from metadata GID %s to actual download GID: %s", t.GID, status.FollowedBy[0])
					t.GID = status.FollowedBy[0]
					continue
				}
				logger.Infof("Download completed for GID %s", t.GID)
				return nil
			}

			// Check for errors
			if status.IsDownloadError() {
				return fmt.Errorf("aria2 download error: %s (code: %s)", status.ErrorMessage, status.ErrorCode)
			}

			if status.IsDownloadRemoved() {
				return errors.New("aria2 download was removed")
			}
		}
	}
}

// getStatus retrieves the current status of the download
func (t *Task) getStatus(ctx context.Context) (*aria2.Status, error) {
	logger := log.FromContext(ctx)

	// Try active/waiting queue first
	status, err := t.Aria2Client.TellStatus(ctx, t.GID)
	if err == nil {
		return status, nil
	}

	// Check stopped queue
	logger.Debugf("Task not in active queue, checking stopped queue")
	stoppedTasks, stopErr := t.Aria2Client.TellStopped(ctx, -1, 100)
	if stopErr != nil {
		return nil, fmt.Errorf("failed to get aria2 status: %w", err)
	}

	for _, task := range stoppedTasks {
		if task.GID == t.GID {
			logger.Debugf("Found task in stopped queue with status: %s", task.Status)
			return &task, nil
		}
	}

	return nil, fmt.Errorf("task GID %s not found: %w", t.GID, err)
}

// transferFiles transfers downloaded files from aria2 to storage
func (t *Task) transferFiles(ctx context.Context) error {
	logger := log.FromContext(ctx)

	status, err := t.Aria2Client.TellStatus(ctx, t.GID)
	if err != nil {
		return fmt.Errorf("failed to get final status: %w", err)
	}

	if len(status.Files) == 0 {
		return errors.New("no files in aria2 download")
	}

	logger.Infof("Transferring %d file(s) to storage %s", len(status.Files), t.Storage.Name())
	transferredCount := 0

	for _, file := range status.Files {
		if file.Selected != "true" {
			logger.Debugf("Skipping unselected file: %s", file.Path)
			continue
		}

		fileName := filepath.Base(file.Path)

		// Skip torrent metadata files
		if filepath.Ext(fileName) == ".torrent" {
			logger.Debugf("Skipping torrent metadata file: %s", fileName)
			t.removeFileIfNeeded(file.Path)
			continue
		}

		if err := t.transferFile(ctx, file.Path); err != nil {
			return err
		}

		transferredCount++
		t.removeFileIfNeeded(file.Path)
	}

	if transferredCount == 0 {
		return errors.New("no files were transferred")
	}

	return nil
}

// transferFile transfers a single file to storage
func (t *Task) transferFile(ctx context.Context, filePath string) error {
	logger := log.FromContext(ctx)

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warnf("Downloaded file not found: %s", filePath)
			return nil // Not a fatal error, continue with other files
		}
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	// Open file
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()

	// Set content length in context for storage
	ctx = context.WithValue(ctx, ctxkey.ContentLength, fileInfo.Size())

	// Save to storage
	fileName := filepath.Base(filePath)
	destPath := filepath.Join(t.StorPath, fileName)

	logger.Infof("Transferring file %s to %s:%s", fileName, t.Storage.Name(), destPath)

	if err := t.Storage.Save(ctx, f, destPath); err != nil {
		return fmt.Errorf("failed to save file %s to storage: %w", fileName, err)
	}

	logger.Infof("Successfully transferred file %s", fileName)
	return nil
}

// removeFileIfNeeded removes a file if RemoveAfterTransfer is enabled
func (t *Task) removeFileIfNeeded(filePath string) {
	if config.C().Aria2.KeepFile {
		return
	}

	logger := log.FromContext(t.ctx)
	if err := os.Remove(filePath); err != nil {
		logger.Warnf("Failed to remove local file %s: %v", filePath, err)
	} else {
		logger.Debugf("Removed local file %s", filePath)
	}
}

// cancelAria2Download cancels the aria2 download task
func (t *Task) cancelAria2Download() {
	logger := log.FromContext(t.ctx)
	logger.Infof("Canceling aria2 download GID: %s", t.GID)

	// Use a background context with timeout for cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to force remove the download
	if _, err := t.Aria2Client.ForceRemove(ctx, t.GID); err != nil {
		logger.Warnf("Failed to cancel aria2 download %s: %v", t.GID, err)
	} else {
		logger.Infof("Successfully canceled aria2 download %s", t.GID)
	}

	// Also remove the download result to clean up
	if _, err := t.Aria2Client.RemoveDownloadResult(ctx, t.GID); err != nil {
		logger.Debugf("Failed to remove download result for %s: %v", t.GID, err)
	}
}
