package batchtfile

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/tdler"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/common/utils/ioutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
)

// downloadToCache fetches elem.File into the element cache path. It resumes
// from a partial .part download tracked by a resume bitmap, and reuses a
// complete cache file (e.g. when the previous run was interrupted during
// upload).
func (t *Task) downloadToCache(ctx context.Context, elem *TaskElement) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", elem.File.Name()))
	if elem.File.Size() > 0 {
		if stat, err := os.Stat(elem.localPath); err == nil && stat.Size() == elem.File.Size() {
			logger.Info("Cache file already complete, skipping download")
			return nil
		}
	}
	onProgress := t.downloadCallback(ctx, elem)
	if elem.File.Size() <= 0 {
		// Unknown size (e.g. photos) cannot be resumed; use the plain downloader.
		localFile, err := fsutil.CreateFile(elem.localPath)
		if err != nil {
			return fmt.Errorf("failed to create local file: %w", err)
		}
		defer localFile.Close()
		wrAt := ioutil.NewProgressWriterAt(localFile, onProgress)
		if _, err := tdler.NewDownloader(elem.File).Parallel(ctx, wrAt); err != nil {
			return err
		}
		return nil
	}
	partPath := elem.localPath + ".part"
	// 不截断已存在的 .part: 位图标记的已完成块依赖既有字节。
	localFile, err := os.OpenFile(partPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	wrAt := ioutil.NewProgressWriterAt(localFile, onProgress)
	err = tdler.DownloadResumable(
		ctx, elem.File, wrAt,
		dlutil.BestThreads(elem.File.Size(), config.C().Threads),
		tdler.ResumeStatePath(partPath),
	)
	closeErr := localFile.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close cache file: %w", closeErr)
	}
	stat, err := os.Stat(partPath)
	if err != nil {
		return fmt.Errorf("failed to stat downloaded file: %w", err)
	}
	if stat.Size() != elem.File.Size() {
		return fmt.Errorf("downloaded size %d does not match expected %d", stat.Size(), elem.File.Size())
	}
	if err := os.Rename(partPath, elem.localPath); err != nil {
		return fmt.Errorf("failed to finalize download: %w", err)
	}
	// 清理位图是尽力而为: 下载已完成, 清理失败不应使任务失败。
	if err := tdler.RemoveResumeState(tdler.ResumeStatePath(partPath)); err != nil {
		logger.Warnf("Failed to remove resume state: %v", err)
	}
	return nil
}

func (t *Task) downloadCallback(ctx context.Context, elem *TaskElement) func(int) {
	return func(n int) {
		t.recordItemDownload(elem.ID, int64(n), time.Now())
		downloaded := t.downloaded.Add(int64(n))
		t.notifyProgress(ctx)
		taskevent.Emit(ctx, taskevent.Event{
			TaskID:          t.ID,
			Phase:           taskevent.PhaseProgress,
			TotalBytes:      t.totalSize,
			DownloadedBytes: downloaded,
		})
	}
}
