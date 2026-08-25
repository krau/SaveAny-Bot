package tfile

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/tdler"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/config"
)

// download fetches the file into the cache path. It resumes from a partial
// .part download tracked by a resume bitmap, and reuses a complete cache
// file (e.g. when the previous run was interrupted during upload).
func (t *Task) download(ctx context.Context) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", t.File.Name()))
	if t.File.Size() > 0 {
		if stat, err := os.Stat(t.localPath); err == nil && stat.Size() == t.File.Size() {
			logger.Info("Cache file already complete, skipping download")
			return nil
		}
	}
	if t.File.Size() <= 0 {
		// Unknown size (e.g. photos) cannot be resumed; use the plain downloader.
		localFile, err := fsutil.CreateFile(t.localPath)
		if err != nil {
			return fmt.Errorf("failed to create local file: %w", err)
		}
		defer localFile.Close()
		wrAt := newWriterAt(ctx, localFile, t.Progress, t)
		if _, err := tdler.NewDownloader(t.File).Parallel(ctx, wrAt); err != nil {
			return err
		}
		logger.Info("File downloaded successfully")
		return nil
	}
	partPath := t.localPath + ".part"
	// 不截断已存在的 .part: 位图标记的已完成块依赖既有字节。
	localFile, err := os.OpenFile(partPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	wrAt := newWriterAt(ctx, localFile, t.Progress, t)
	err = tdler.DownloadResumable(
		ctx, t.File, wrAt,
		dlutil.BestThreads(t.File.Size(), config.C().Threads),
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
	if stat.Size() != t.File.Size() {
		return fmt.Errorf("downloaded size %d does not match expected %d", stat.Size(), t.File.Size())
	}
	if err := os.Rename(partPath, t.localPath); err != nil {
		return fmt.Errorf("failed to finalize download: %w", err)
	}
	// 清理位图是尽力而为: 下载已完成, 清理失败不应使任务失败。
	if err := tdler.RemoveResumeState(tdler.ResumeStatePath(partPath)); err != nil {
		logger.Warnf("Failed to remove resume state: %v", err)
	}
	logger.Info("File downloaded successfully")
	return nil
}
