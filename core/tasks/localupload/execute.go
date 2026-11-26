package localupload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"

	"github.com/krau/SaveAny-Bot/common/upload"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
)

const progressInterval = 200 * time.Millisecond

func (t *Task) Execute(ctx context.Context) (retErr error) {
	defer func() {
		t.setResult(retErr)
		if retErr != nil && !isCancelled(retErr) {
			if err := upload.RemoveStaging(t.ID); err != nil {
				log.FromContext(ctx).Warnf("Failed to remove failed staged upload %s: %v", t.ID, err)
			}
		}
	}()

	sourcePath, err := upload.StagingPayloadPath(t.ID)
	if err != nil {
		return err
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat staged upload: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("staged upload is not a regular file")
	}
	if t.Size > 0 && info.Size() != t.Size {
		return fmt.Errorf("staged upload size is %d bytes, expected %d", info.Size(), t.Size)
	}
	t.Size = info.Size()

	taskevent.Emit(ctx, taskevent.Event{
		TaskID:     t.ID,
		Phase:      taskevent.PhaseStart,
		TotalBytes: t.Size,
		TotalFiles: 1,
	})
	var lastEmit atomic.Int64
	onProgress := func(uploaded, total int64) {
		if total <= 0 {
			total = t.Size
		}
		t.uploaded.Store(uploaded)
		now := time.Now().UnixNano()
		last := lastEmit.Load()
		if uploaded < total && now-last < int64(progressInterval) {
			return
		}
		if lastEmit.CompareAndSwap(last, now) {
			emitProgress(ctx, t.ID, total, uploaded)
		}
	}

	targetPath := path.Join(t.Path, t.Name)
	err = upload.SaveFile(ctx, sourcePath, targetPath, upload.Options{
		Storage:    t.Storage,
		Overwrite:  t.Overwrite,
		Retries:    max(config.C().Retry, 1),
		RetryDelay: 3 * time.Second,
		OnProgress: onProgress,
	})
	if err != nil {
		return fmt.Errorf("failed to upload staged file: %w", err)
	}

	t.uploaded.Store(t.Size)
	emitProgress(ctx, t.ID, t.Size, t.Size)
	if err := upload.RemoveStaging(t.ID); err != nil {
		log.FromContext(ctx).Warnf("Failed to remove staged upload %s: %v", t.ID, err)
	}
	return nil
}

func emitProgress(ctx context.Context, taskID string, total, uploaded int64) {
	taskevent.Emit(ctx, taskevent.Event{
		TaskID:          taskID,
		Phase:           taskevent.PhaseProgress,
		TotalBytes:      total,
		DownloadedBytes: uploaded,
		TotalFiles:      1,
		DownloadedFiles: boolToInt(uploaded >= total && total > 0),
	})
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func isCancelled(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
