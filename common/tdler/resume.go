package tdler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"golang.org/x/sync/errgroup"

	"github.com/krau/SaveAny-Bot/common/utils/rpclimit"
	"github.com/krau/SaveAny-Bot/pkg/consts/tglimit"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

const maxChunkRetries = 20

// errCDNRedirect reports a chunk served from a CDN. Raw upload.getFile
// requests cannot follow CDN redirects (they need gotd's re-hash dance), so
// the download falls back to the plain downloader, which handles them.
var errCDNRedirect = errors.New("CDN redirect")

func partPathOf(bitmapPath string) string {
	return strings.TrimSuffix(bitmapPath, ".bitmap")
}

// resumeBitmap records which partSize-aligned blocks of a download have been
// durably written, so an interrupted download can continue after a restart.
// The bitmap file is rewritten atomically after every completed block.
type resumeBitmap struct {
	PartSize         int      `json:"part_size"`
	Size             int64    `json:"size"`
	Blocks           []uint64 `json:"blocks"`
	SourceKind       string   `json:"source_kind,omitempty"`
	SourceID         int64    `json:"source_id,omitempty"`
	SourceAccessHash int64    `json:"source_access_hash,omitempty"`

	mu       sync.Mutex
	dirty    int
	lastSave time.Time
}

func newResumeBitmap(size int64, identity tfile.FileIdentity) *resumeBitmap {
	b := &resumeBitmap{
		PartSize:         tglimit.MaxPartSize,
		Size:             size,
		SourceKind:       identity.Kind,
		SourceID:         identity.ID,
		SourceAccessHash: identity.AccessHash,
	}
	b.ensureBlocks()
	return b
}

func (b *resumeBitmap) matches(identity tfile.FileIdentity) bool {
	if b.SourceKind == "" && b.SourceID == 0 && b.SourceAccessHash == 0 {
		// State written by an earlier version without identity metadata.
		return true
	}
	return b.SourceKind == identity.Kind && b.SourceID == identity.ID && b.SourceAccessHash == identity.AccessHash
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

// coveredBytes returns the byte range the bitmap claims is durably written:
// through the highest completed block, capped at the expected size.
func (b *resumeBitmap) coveredBytes() int64 {
	highest := -1
	for word := range b.Blocks {
		if b.Blocks[word] == 0 {
			continue
		}
		bit := 63 - bits.LeadingZeros64(b.Blocks[word])
		if block := word*64 + bit; block > highest {
			highest = block
		}
	}
	if highest < 0 {
		return 0
	}
	covered := int64(highest+1) * int64(b.PartSize)
	return min(covered, b.Size)
}

// maxResumableSize bounds the bitmap allocation derived from a (possibly
// corrupt) size field: 1 TiB needs ~16 KiB of bitmap words, far above any
// real Telegram file.
const maxResumableSize int64 = 1 << 40

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
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return nil, fmt.Errorf("remove invalid resume bitmap: %w", removeErr)
		}
		return nil, nil
	}
	// 位图只写 PartSize=MaxPartSize; 其他值或超大 size 都是损坏数据,
	// 在按 size 分配内存前拒绝, 防止构造的位图触发巨量分配。
	if b.PartSize != tglimit.MaxPartSize || b.Size > maxResumableSize {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return nil, fmt.Errorf("remove incompatible resume bitmap: %w", removeErr)
		}
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
	b.dirty++
	// Persist at most once per second or every 16 completed blocks. A crash
	// between saves only re-downloads the unwritten tail; it cannot mark
	// missing data as complete.
	if b.dirty < 16 && !b.lastSave.IsZero() && time.Since(b.lastSave) < time.Second {
		return nil
	}
	if err := b.saveLocked(path); err != nil {
		return err
	}
	b.dirty = 0
	b.lastSave = time.Now()
	return nil
}

func (b *resumeBitmap) flush(path string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.dirty == 0 {
		return nil
	}
	if err := b.saveLocked(path); err != nil {
		return err
	}
	b.dirty = 0
	b.lastSave = time.Now()
	return nil
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
func fetchChunk(
	ctx context.Context,
	client downloader.Client,
	location tg.InputFileLocationClass,
	offset int64,
	limit int,
) ([]byte, error) {
	req := &tg.UploadGetFileRequest{
		Location: location,
		Offset:   offset,
		Limit:    limit,
	}
	timeoutRetries := 0
	for {
		res, err := client.UploadGetFile(ctx, req)
		if err == nil {
			switch r := res.(type) {
			case *tg.UploadFile:
				if len(r.Bytes) > limit {
					return nil, fmt.Errorf("get chunk at %d: server returned %d bytes, limit %d", offset, len(r.Bytes), limit)
				}
				return r.Bytes, nil
			case *tg.UploadFileCDNRedirect:
				return nil, fmt.Errorf("%w (dc %d)", errCDNRedirect, r.DCID)
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
	if threads < 1 {
		threads = 1
	}
	bm, err := loadResumeBitmap(bitmapPath)
	if err != nil {
		return err
	}
	if bm != nil && !bm.matches(tfile.Identity(file.Location())) {
		if err := os.Remove(bitmapPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove stale resume bitmap: %w", err)
		}
		bm = nil
	}
	// 位图描述的数据文件 (bitmapPath 去掉 .bitmap 后缀) 必须存在且非空,
	// 且至少覆盖位图标记为完成的最大块范围: 若缺失、为空或更短,
	// 已标记完成的块字节已丢失, 必须重置位图全量重下 (否则 WriteAt 会
	// 在文件中留下零字节洞, 而最终尺寸校验恰好通过, 造成静默损坏)。
	if bm != nil {
		if stat, err := os.Stat(partPathOf(bitmapPath)); err != nil || stat.Size() < bm.coveredBytes() {
			if err := os.Remove(bitmapPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("reset stale resume bitmap: %w", err)
			}
			bm = nil
		}
	}
	if bm == nil || bm.PartSize != tglimit.MaxPartSize || bm.Size != file.Size() {
		// 位图缺失或不兼容: 截断旧的部分文件, 避免残留尾部字节导致最终大小校验失败。
		if err := os.Truncate(partPathOf(bitmapPath), 0); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("truncate stale part file: %w", err)
		}
		bm = newResumeBitmap(file.Size(), tfile.Identity(file.Location()))
		if err := bm.save(bitmapPath); err != nil {
			return err
		}
	}
	missing := bm.missingBlocks()
	if len(missing) == 0 {
		return nil
	}

	client := rpclimit.WrapDownloader(file.Dler())
	location := file.Location()
	var writeMu sync.Mutex
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(threads)
	for _, block := range missing {
		if err := gctx.Err(); err != nil {
			break
		}
		eg.Go(func() error {
			offset := int64(block) * int64(bm.PartSize)
			data, err := fetchChunk(gctx, client, location, offset, bm.PartSize)
			if err != nil {
				return err
			}
			want := int64(bm.PartSize)
			if offset+want > bm.Size {
				want = bm.Size - offset
			}
			if int64(len(data)) != want {
				return fmt.Errorf("chunk at offset %d has size %d, want %d", offset, len(data), want)
			}
			writeMu.Lock()
			defer writeMu.Unlock()
			n, err := w.WriteAt(data, offset)
			if err != nil {
				return fmt.Errorf("write chunk at offset %d: %w", offset, err)
			}
			if n != len(data) {
				return fmt.Errorf("write chunk at offset %d: short write %d/%d", offset, n, len(data))
			}
			return bm.markAndSave(block, bitmapPath)
		})
	}
	err = eg.Wait()
	if flushErr := bm.flush(bitmapPath); flushErr != nil {
		return flushErr
	}
	if err != nil {
		if errors.Is(err, errCDNRedirect) {
			// CDN 托管的文件无法用裸 upload.getFile 续传, 回退到 gotd 下载器
			// (它内部处理 CDN 重定向), 全量重下本文件。
			log.FromContext(ctx).Warn("File served from CDN, falling back to plain downloader")
			if _, err := newDownloader(file).Parallel(ctx, w); err != nil {
				return fmt.Errorf("plain download after CDN redirect: %w", err)
			}
			return nil
		}
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
