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

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var status *aria2.Status
	var err error

	for {
		select {
		case <-ctx.Done():
			logger.Warn("Aria2 task canceled")
			if t.Progress != nil {
				t.Progress.OnDone(ctx, t, ctx.Err())
			}
			return ctx.Err()
		case <-ticker.C:
			// Try to get status from active/waiting queue first
			status, err = t.Aria2Client.TellStatus(ctx, t.GID)
			if err != nil {
				// If GID not found in active queue, check stopped queue
				logger.Debugf("Task not in active queue, checking stopped queue: %v", err)
				stoppedTasks, stopErr := t.Aria2Client.TellStopped(ctx, -1, 100)
				if stopErr != nil {
					logger.Errorf("Failed to get stopped tasks: %v", stopErr)
					if t.Progress != nil {
						t.Progress.OnDone(ctx, t, err)
					}
					return fmt.Errorf("failed to get aria2 status: %w", err)
				}

				// Find our task in stopped queue
				found := false
				for _, task := range stoppedTasks {
					if task.GID == t.GID {
						status = &task
						found = true
						logger.Debugf("Found task in stopped queue with status: %s", status.Status)
						break
					}
				}

				if !found {
					logger.Errorf("Task GID %s not found in active or stopped queue", t.GID)
					if t.Progress != nil {
						t.Progress.OnDone(ctx, t, err)
					}
					return fmt.Errorf("aria2 task not found: %w", err)
				}
			}

			logger.Debugf("Aria2 GID %s status: %s, completed: %s/%s",
				t.GID, status.Status, status.CompletedLength, status.TotalLength)

			if t.Progress != nil {
				t.Progress.OnProgress(ctx, t, status)
			}

			// Check if download is complete
			if status.IsDownloadComplete() {
				logger.Infof("Aria2 download completed for GID %s", t.GID)
				goto TransferFiles
			}

			// Check for errors
			if status.IsDownloadError() {
				err := fmt.Errorf("aria2 download error: %s (code: %s)", status.ErrorMessage, status.ErrorCode)
				logger.Errorf("Aria2 download failed: %v", err)
				if t.Progress != nil {
					t.Progress.OnDone(ctx, t, err)
				}
				return err
			}

			// Check if removed
			if status.IsDownloadRemoved() {
				err := errors.New("aria2 download was removed")
				logger.Error("Aria2 download was removed")
				if t.Progress != nil {
					t.Progress.OnDone(ctx, t, err)
				}
				return err
			}
		}
	}

TransferFiles:
	// Get final status to get file list
	status, err = t.Aria2Client.TellStatus(ctx, t.GID)
	if err != nil {
		logger.Errorf("Failed to get final status: %v", err)
		if t.Progress != nil {
			t.Progress.OnDone(ctx, t, err)
		}
		return fmt.Errorf("failed to get final status: %w", err)
	}

	if len(status.Files) == 0 {
		err := errors.New("no files in aria2 download")
		logger.Error("No files in aria2 download")
		if t.Progress != nil {
			t.Progress.OnDone(ctx, t, err)
		}
		return err
	}

	// Transfer files to storage
	logger.Infof("Transferring %d file(s) to storage %s", len(status.Files), t.Storage.Name())
	for _, file := range status.Files {
		if file.Selected != "true" {
			logger.Debugf("Skipping unselected file: %s", file.Path)
			continue
		}

		// Check if file exists
		if _, err := os.Stat(file.Path); os.IsNotExist(err) {
			logger.Errorf("Downloaded file not found: %s", file.Path)
			continue
		}

		// Open file
		f, err := os.Open(file.Path)
		if err != nil {
			logger.Errorf("Failed to open file %s: %v", file.Path, err)
			if t.Progress != nil {
				t.Progress.OnDone(ctx, t, err)
			}
			return fmt.Errorf("failed to open file %s: %w", file.Path, err)
		}
		defer f.Close()

		// Get file info
		fileInfo, err := f.Stat()
		if err != nil {
			logger.Errorf("Failed to stat file %s: %v", file.Path, err)
			if t.Progress != nil {
				t.Progress.OnDone(ctx, t, err)
			}
			return fmt.Errorf("failed to stat file %s: %w", file.Path, err)
		}

		// Set content length in context for storage
		ctx = context.WithValue(ctx, ctxkey.ContentLength, fileInfo.Size())

		// Determine destination path
		fileName := filepath.Base(file.Path)
		destPath := filepath.Join(t.StorPath, fileName)

		logger.Infof("Transferring file %s to %s:%s", fileName, t.Storage.Name(), destPath)

		// Save to storage
		err = t.Storage.Save(ctx, f, destPath)
		if err != nil {
			logger.Errorf("Failed to save file %s to storage: %v", fileName, err)
			if t.Progress != nil {
				t.Progress.OnDone(ctx, t, err)
			}
			return fmt.Errorf("failed to save file %s to storage: %w", fileName, err)
		}

		logger.Infof("Successfully transferred file %s", fileName)

		// Optionally remove the local file after successful transfer
		if config.C().Aria2.RemoveAfterTransfer {
			if err := os.Remove(file.Path); err != nil {
				logger.Warnf("Failed to remove local file %s: %v", file.Path, err)
			} else {
				logger.Debugf("Removed local file %s", file.Path)
			}
		}
	}

	logger.Infof("Aria2 task %s completed successfully", t.ID)
	if t.Progress != nil {
		t.Progress.OnDone(ctx, t, nil)
	}

	// Clean up aria2 download result
	_, err = t.Aria2Client.RemoveDownloadResult(ctx, t.GID)
	if err != nil {
		logger.Warnf("Failed to remove aria2 download result: %v", err)
	}

	return nil
}
