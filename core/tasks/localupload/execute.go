package localupload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/krau/SaveAny-Bot/common/upload"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
)

const progressInterval = 200 * time.Millisecond

func (t *Task) Execute(ctx context.Context) (retErr error) {
	var actualPath string
	defer func() { t.setResult(ctx, actualPath, retErr) }()

	sourcePath, err := upload.StagingPayloadPath(t.stagingRoot, t.ID)
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
	if info.Size() != t.Size {
		return fmt.Errorf("staged upload size is %d bytes, expected %d", info.Size(), t.Size)
	}

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
			emitProgress(ctx, t.ID, total, uploaded, 0)
		}
	}

	actualPath, err = upload.SaveFileWithPath(ctx, sourcePath, t.StoragePath(), upload.Options{
		Storage:       t.Storage,
		Overwrite:     t.Overwrite,
		Retries:       max(config.C().Retry, 1),
		RetryDelay:    3 * time.Second,
		OnProgress:    onProgress,
		RequireUpload: true,
	})
	if err != nil {
		return fmt.Errorf("failed to upload staged file: %w", err)
	}

	t.uploaded.Store(t.Size)
	emitProgress(ctx, t.ID, t.Size, t.Size, 1)
	return nil
}

func emitProgress(ctx context.Context, taskID string, total, uploaded int64, completedFiles int) {
	taskevent.Emit(ctx, taskevent.Event{
		TaskID:          taskID,
		Phase:           taskevent.PhaseProgress,
		TotalBytes:      total,
		DownloadedBytes: uploaded,
		TotalFiles:      1,
		DownloadedFiles: completedFiles,
	})
}
