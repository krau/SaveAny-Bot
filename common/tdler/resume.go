package tdler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"golang.org/x/sync/errgroup"

	"github.com/krau/SaveAny-Bot/pkg/consts/tglimit"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

const maxChunkRetries = 20

// resumeBitmap records which partSize-aligned blocks of a download have been
// durably written, so an interrupted download can continue after a restart.
// The bitmap file is rewritten atomically after every completed block.
type resumeBitmap struct {
	PartSize int      `json:"part_size"`
	Size     int64    `json:"size"`
	Blocks   []uint64 `json:"blocks"`

	mu sync.Mutex
}

func newResumeBitmap(size int64) *resumeBitmap {
	b := &resumeBitmap{PartSize: tglimit.MaxPartSize, Size: size}
	b.ensureBlocks()
	return b
}

func (b *resumeBitmap) ensureBlocks() {
	if need := (b.blockCount() + 63) / 64; len(b.Blocks) < need {
		b.Blocks = make([]uint64, need)
	}
}

func (b *resumeBitmap) blockCount() int {
	return int((b.Size + int64(b.PartSize) - 1) / int64(b.PartSize))
}

func (b *resumeBitmap) isDone(block int) bool {
	return b.Blocks[block/64]&(1<<uint(block%64)) != 0
}

func (b *resumeBitmap) markDone(block int) {
	b.Blocks[block/64] |= 1 << uint(block%64)
}

func (b *resumeBitmap) complete() bool {
	for block := 0; block < b.blockCount(); block++ {
		if !b.isDone(block) {
			return false
		}
	}
	return true
}

func (b *resumeBitmap) missingBlocks() []int {
	missing := make([]int, 0, b.blockCount())
	for block := 0; block < b.blockCount(); block++ {
		if !b.isDone(block) {
			missing = append(missing, block)
		}
	}
	return missing
}

func loadResumeBitmap(path string) (*resumeBitmap, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read resume bitmap: %w", err)
	}
	var b resumeBitmap
	if err := json.Unmarshal(data, &b); err != nil {
		// 无法解析的位图 (外部损坏): 删除并视为不存在, 全量重下自愈。
		_ = os.Remove(path)
		return nil, nil
	}
	if b.Size <= 0 || b.PartSize <= 0 {
		// 无效位图 (损坏或旧格式), 视为不存在, 全量重下。
		_ = os.Remove(path)
		return nil, nil
	}
	b.ensureBlocks()
	return &b, nil
}

func (b *resumeBitmap) save(path string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.saveLocked(path)
}

func (b *resumeBitmap) saveLocked(path string) error {
	data, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("marshal resume bitmap: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write resume bitmap: %w", err)
	}
	return os.Rename(tmp, path)
}

func (b *resumeBitmap) markAndSave(block int, path string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.markDone(block)
	return b.saveLocked(path)
}

func isRetryableTimeout(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	if tgerr.Is(err, tg.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// fetchChunk downloads one partSize-aligned chunk, retrying flood waits and
// transient timeouts like gotd's downloader does.
func fetchChunk(ctx context.Context, file tfile.TGFile, offset int64, limit int) ([]byte, error) {
	req := &tg.UploadGetFileRequest{
		Location: file.Location(),
		Offset:   offset,
		Limit:    limit,
	}
	timeoutRetries := 0
	for {
		res, err := file.Dler().UploadGetFile(ctx, req)
		if err == nil {
			switch r := res.(type) {
			case *tg.UploadFile:
				return r.Bytes, nil
			case *tg.UploadFileCDNRedirect:
				return nil, fmt.Errorf("CDN redirect is not supported (dc %d)", r.DCID)
			default:
				return nil, fmt.Errorf("unexpected upload.getFile response %T", res)
			}
		}
		if flood, ferr := tgerr.FloodWait(ctx, err); ferr != nil {
			if flood {
				// FloodWait already slept; retry.
				continue
			}
			if isRetryableTimeout(ctx, ferr) {
				timeoutRetries++
				if timeoutRetries >= maxChunkRetries {
					return nil, fmt.Errorf("get chunk at %d: retry limit reached: %w", offset, ferr)
				}
				continue
			}
			return nil, fmt.Errorf("get chunk at %d: %w", offset, ferr)
		}
	}
}

// DownloadResumable downloads file to w in partSize chunks, skipping blocks
// already recorded as complete in bitmapPath and persisting every completed
// block so an interrupted download can resume. A missing or incompatible
// bitmap starts a full download. Requires a known, non-zero file size.
func DownloadResumable(
	ctx context.Context,
	file tfile.TGFile,
	w io.WriterAt,
	threads int,
	bitmapPath string,
) error {
	if file.Size() <= 0 {
		return fmt.Errorf("resumable download requires a known size")
	}
	bm, err := loadResumeBitmap(bitmapPath)
	if err != nil {
		return err
	}
	// 位图描述的数据文件 (bitmapPath 去掉 .bitmap 后缀) 必须存在且非空:
	// 若缺失或为空, 已标记完成的块字节已丢失, 必须重置位图全量重下。
	if bm != nil {
		partPath := strings.TrimSuffix(bitmapPath, ".bitmap")
		if stat, err := os.Stat(partPath); err != nil || stat.Size() == 0 {
			if err := os.Remove(bitmapPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("reset stale resume bitmap: %w", err)
			}
			bm = nil
		}
	}
	if bm == nil || bm.PartSize != tglimit.MaxPartSize || bm.Size != file.Size() {
		bm = newResumeBitmap(file.Size())
		if err := bm.save(bitmapPath); err != nil {
			return err
		}
	}
	missing := bm.missingBlocks()
	if len(missing) == 0 {
		return nil
	}

	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(threads)
	for _, block := range missing {
		block := block
		eg.Go(func() error {
			offset := int64(block) * int64(bm.PartSize)
			data, err := fetchChunk(gctx, file, offset, bm.PartSize)
			if err != nil {
				return err
			}
			if len(data) == 0 {
				return fmt.Errorf("file ended early at offset %d (expected size %d)", offset, bm.Size)
			}
			if _, err := w.WriteAt(data, offset); err != nil {
				return fmt.Errorf("write chunk at offset %d: %w", offset, err)
			}
			return bm.markAndSave(block, bitmapPath)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	if !bm.complete() {
		return fmt.Errorf("download finished with missing blocks")
	}
	return nil
}

// RemoveResumeState deletes the bitmap file of a completed download.
func RemoveResumeState(bitmapPath string) error {
	if err := os.Remove(bitmapPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove resume bitmap: %w", err)
	}
	if err := os.Remove(bitmapPath + ".tmp"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove resume bitmap temp: %w", err)
	}
	return nil
}

// ResumeStatePath returns the bitmap path for a download cache file.
func ResumeStatePath(cachePath string) string {
	return cachePath + ".bitmap"
}
