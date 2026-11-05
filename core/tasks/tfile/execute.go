package tfile

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/retry"
	"github.com/krau/SaveAny-Bot/common/tdler"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/common/utils/ioutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	tfilepkg "github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

func (t *Task) Execute(ctx context.Context) (err error) {
	defer func() { t.execErr = err }()
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", t.File.Name()))
	if t.Progress != nil {
		t.Progress.OnStart(ctx, t)
	}
	if t.stream {
		return executeStream(ctx, t)
	}

	logger.Info("Starting file download")
	defer func() {
		if t.Progress != nil {
			t.Progress.OnDone(ctx, t, err)
		}
	}()
	if err := tdler.DownloadToCache(ctx, t.File, t.localPath, func(w io.WriterAt) io.WriterAt {
		return newWriterAt(ctx, w, t.Progress, t)
	}); err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	logger.Infof("File downloaded successfully")
	if path.Ext(t.File.Name()) == "" && path.Ext(t.Path) == "" {
		ext := fsutil.DetectFileExt(t.localPath)
		if ext != "" {
			t.Path = t.Path + ext
		}
	}
	fileStat, err := os.Stat(t.localPath)
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}
	if t.File.Size() > 0 && fileStat.Size() == 0 {
		return fmt.Errorf("downloaded file is empty (expected %d bytes)", t.File.Size())
	}
	vctx := context.WithValue(ctx, ctxkey.ContentLength, fileStat.Size())
	vctx = storagetypes.WithMessageMetadata(vctx, t.metadataJSON)
	if caption, ok := sourceCaption(t.File); ok {
		vctx = storagetypes.WithSourceCaption(vctx, caption)
	}
	savedPath := t.Path
	err = retry.Retry(func() error {
		file, err := os.Open(t.localPath)
		if err != nil {
			return fmt.Errorf("failed to open cache file: %w", err)
		}
		defer file.Close()
		uploadProgress, tracksUpload := t.Progress.(UploadProgressTracker)
		if !tracksUpload {
			if savedPath, err = storage.SaveWithPath(vctx, t.Storage, file, t.Path); err != nil {
				return fmt.Errorf("failed to save file: %w", err)
			}
			return nil
		}

		uploadProgress.OnUploadStart(vctx, t, fileStat.Size())
		onProgress := func(uploaded, total int64) {
			uploadProgress.OnUploadProgress(vctx, t, uploaded, total)
		}
		if progressSaver, ok := t.Storage.(storage.StorageProgressSaver); ok {
			err = progressSaver.SaveWithProgress(vctx, file, t.Path, onProgress)
			savedPath = t.Path
		} else {
			var reader io.Reader = ioutil.NewProgressReader(file, fileStat.Size(), onProgress)
			savedPath, err = storage.SaveWithPath(vctx, t.Storage, reader, t.Path)
		}
		if err != nil {
			return fmt.Errorf("failed to save file: %w", err)
		}
		return nil
	}, retry.RetryTimes(uint(config.C().Retry)), retry.Context(vctx))
	if err != nil {
		return fmt.Errorf("failed to save file after retries: %w", err)
	}
	if err := core.MarkTaskElementDone(ctx, t.ID, t.ElementID); err != nil {
		return fmt.Errorf("failed to persist task completion: %w", err)
	}
	t.saveMetadataSidecar(ctx, savedPath)
	if err := os.Remove(t.localPath); err != nil {
		logger.Errorf("Failed to remove cache file: %v", err)
	}
	return nil
}

func sourceCaption(file tfilepkg.TGFile) (string, bool) {
	messageFile, ok := file.(tfilepkg.TGFileMessage)
	if !ok || messageFile.Message() == nil {
		return "", false
	}
	return messageFile.Message().GetMessage(), true
}
