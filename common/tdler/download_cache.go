package tdler

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"

	"github.com/krau/SaveAny-Bot/client/relay"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

// DownloadToCache downloads file into cachePath, reporting download progress
// through wrap. A complete cache file is reused as-is (e.g. when the previous
// run was interrupted during upload). Downloads with a known size resume from
// a partial .part file tracked by a resume bitmap, which is removed once the
// download is finalized. Files with an unknown size (e.g. photos) are fetched
// with the plain downloader.
func DownloadToCache(
	ctx context.Context,
	file tfile.TGFile,
	cachePath string,
	wrap func(io.WriterAt) io.WriterAt,
) error {
	if cachePath == "" {
		return fmt.Errorf("empty cache path")
	}
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", file.Name()))
	if file.Size() > 0 {
		if stat, err := os.Stat(cachePath); err == nil && stat.Size() == file.Size() {
			logger.Info("Cache file already complete, skipping download")
			_ = os.Remove(cachePath + ".part")
			relay.CleanupResumeState(ctx, relay.ResumeStatePath(cachePath))
			if err := RemoveResumeState(ResumeStatePath(cachePath + ".part")); err != nil {
				logger.Warnf("Failed to remove stale resume state: %v", err)
			}
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	var cleanup func(context.Context)
	if file.Size() > 0 {
		file, cleanup = relay.PrepareResumable(ctx, file, relay.ResumeStatePath(cachePath))
	} else {
		file, cleanup = relay.Prepare(ctx, file)
	}
	if file.Size() <= 0 {
		if cleanup != nil {
			defer cleanup(ctx)
		}
		// Unknown size (e.g. photos) cannot be resumed; use the plain downloader.
		localFile, err := os.Create(cachePath)
		if err != nil {
			return fmt.Errorf("failed to create cache file: %w", err)
		}
		defer localFile.Close()
		if _, err := newDownloader(file).Parallel(ctx, wrap(localFile)); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
		return nil
	}
	partPath := cachePath + ".part"
	// 不截断已存在的 .part: 位图标记的已完成块依赖既有字节。
	partFile, err := os.OpenFile(partPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open cache file: %w", err)
	}
	downloadErr := DownloadResumable(
		ctx, file, wrap(partFile),
		downloadThreads(file),
		ResumeStatePath(partPath),
	)
	if downloadErr != nil {
		partFile.Close()
		// Keep the relay message and sidecar so a later run can reuse them.
		return downloadErr
	}
	if cleanup != nil {
		defer cleanup(ctx)
	}
	// 落盘后再关闭并 rename: 确保位图标记的字节确实持久, 掉电后不会
	// 出现 "位图完整但数据页丢失" 的零字节洞。
	if err := partFile.Sync(); err != nil {
		partFile.Close()
		return fmt.Errorf("failed to sync cache file: %w", err)
	}
	closeErr := partFile.Close()
	if closeErr != nil {
		return fmt.Errorf("failed to close cache file: %w", closeErr)
	}
	stat, err := os.Stat(partPath)
	if err != nil {
		return fmt.Errorf("failed to stat downloaded file: %w", err)
	}
	if stat.Size() > file.Size() {
		if err := os.Truncate(partPath, file.Size()); err != nil {
			return fmt.Errorf("truncate downloaded file: %w", err)
		}
		stat, err = os.Stat(partPath)
		if err != nil {
			return fmt.Errorf("failed to stat downloaded file: %w", err)
		}
	}
	if stat.Size() != file.Size() {
		return fmt.Errorf("downloaded size %d does not match expected %d", stat.Size(), file.Size())
	}
	if err := os.Rename(partPath, cachePath); err != nil {
		return fmt.Errorf("failed to finalize download: %w", err)
	}
	// 清理位图是尽力而为: 下载已完成, 清理失败不应使任务失败。
	if err := RemoveResumeState(ResumeStatePath(partPath)); err != nil {
		logger.Warnf("Failed to remove resume state: %v", err)
	}
	return nil
}
