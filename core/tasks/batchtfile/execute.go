package batchtfile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/retry"
	"github.com/krau/SaveAny-Bot/common/tdler"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/common/utils/ioutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/pkg/taskevent"
	"github.com/krau/SaveAny-Bot/storage"
	"golang.org/x/sync/errgroup"
)

type executionGroup struct {
	elems      []*TaskElement
	batchSaver storage.StorageBatchSaver
}

func (g executionGroup) usesBatchSaver() bool {
	return g.batchSaver != nil
}

func (t *Task) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("batch_file[%s]", t.ID))
	logger.Info("Starting batch file task")
	if t.Progress != nil {
		t.Progress.OnStart(ctx, t)
	}
	groups := t.executionGroups()
	var err error
	for i := 0; i < len(groups); {
		if groups[i].usesBatchSaver() {
			err = t.processBatch(ctx, groups[i])
			i++
		} else {
			end := i + 1
			for end < len(groups) && !groups[end].usesBatchSaver() {
				end++
			}
			elems := make([]*TaskElement, 0, end-i)
			for _, group := range groups[i:end] {
				elems = append(elems, group.elems...)
			}
			err = t.processElements(ctx, elems)
			i = end
		}
		if err != nil {
			if !t.IgnoreErrors || errors.Is(err, context.Canceled) {
				break
			}
			logger.Warnf("Group processing failed (ignored): %v", err)
			err = nil
		}
	}
	if err != nil {
		logger.Errorf("Error during batch file processing: %v", err)
	} else {
		logger.Info("Batch file task completed successfully")
	}
	t.finishItems(err)
	if t.Progress != nil {
		t.Progress.OnDone(ctx, t, err)
	}
	return err
}

func (t *Task) notifyProgress(ctx context.Context) {
	if t.Progress != nil {
		t.Progress.OnProgress(ctx, t)
	}
}

func (t *Task) executionGroups() []executionGroup {
	groups := make([]executionGroup, 0, len(t.elems))
	for i := 0; i < len(t.elems); {
		elem := &t.elems[i]
		batchSaver, batchCapable := elem.Storage.(storage.StorageBatchSaver)
		key := elem.executionGroupKey()
		if !batchCapable || key == "" {
			groups = append(groups, executionGroup{elems: []*TaskElement{elem}})
			i++
			continue
		}

		end := i + 1
		for end < len(t.elems) {
			next := &t.elems[end]
			if next.Storage != elem.Storage || next.executionGroupKey() != key {
				break
			}
			end++
		}
		elems := make([]*TaskElement, 0, end-i)
		for j := i; j < end; j++ {
			elems = append(elems, &t.elems[j])
		}
		groups = append(groups, executionGroup{elems: elems, batchSaver: batchSaver})
		i = end
	}
	return groups
}

func (t *Task) processElements(ctx context.Context, elems []*TaskElement) error {
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(config.C().Workers)
	for _, elem := range elems {
		eg.Go(func() error {
			if err := t.markProcessing(ctx, elem); err != nil {
				return err
			}
			defer t.unmarkProcessing(elem.ID)
			err := t.processElement(gctx, elem)
			if err != nil && t.IgnoreErrors && !errors.Is(err, context.Canceled) {
				// Per-item failure: keep siblings running.
				log.FromContext(ctx).Warnf("Element %s failed (ignored): %v", elem.ID, err)
				return nil
			}
			return err
		})
	}
	return eg.Wait()
}

func (t *Task) processBatch(ctx context.Context, group executionGroup) error {
	var cleanup []*TaskElement
	defer func() {
		for _, elem := range cleanup {
			if err := os.Remove(elem.localPath); err != nil && !os.IsNotExist(err) {
				log.FromContext(ctx).Warnf("Failed to cleanup batch cache file %s: %v", elem.localPath, err)
			}
		}
	}()

	type downloadResult struct {
		elem *TaskElement
		err  error
	}
	results := make([]downloadResult, len(group.elems))
	var resultsMu sync.Mutex

	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(config.C().Workers)
	for i, elem := range group.elems {
		eg.Go(func() error {
			if err := t.markProcessing(ctx, elem); err != nil {
				return err
			}
			defer t.unmarkProcessing(elem.ID)
			err := t.downloadElement(gctx, elem)
			// Store by original index.
			resultsMu.Lock()
			results[i] = downloadResult{elem: elem, err: err}
			resultsMu.Unlock()
			if err != nil && t.IgnoreErrors && !errors.Is(err, context.Canceled) {
				// Per-item failure: keep siblings running.
				log.FromContext(ctx).Warnf("Element %s failed (ignored): %v", elem.ID, err)
				return nil
			}
			return err
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	successElems := make([]*TaskElement, 0, len(group.elems))
	for _, r := range results {
		if r.err == nil {
			successElems = append(successElems, r.elem)
		}
	}
	if len(successElems) == 0 {
		return fmt.Errorf("all elements failed to download")
	}

	items := make([]storagetypes.BatchItem, 0, len(successElems))
	openFiles := make([]*os.File, 0, len(successElems))
	defer func() {
		for _, file := range openFiles {
			if err := file.Close(); err != nil {
				log.FromContext(ctx).Warnf("Failed to close batch cache file %s: %v", file.Name(), err)
			}
		}
	}()
	for _, elem := range successElems {
		file, err := os.Open(elem.localPath)
		if err != nil {
			t.markItemFailed(ctx, elem.ID, FailureStageCache, err)
			return fmt.Errorf("failed to open cache file: %w", err)
		}
		stat, err := file.Stat()
		if err != nil {
			file.Close()
			t.markItemFailed(ctx, elem.ID, FailureStageCache, err)
			return fmt.Errorf("failed to get cache file stat: %w", err)
		}
		openFiles = append(openFiles, file)
		items = append(items, storagetypes.BatchItem{
			Reader:           file,
			StoragePath:      elem.Path,
			Size:             stat.Size(),
			SourceGroupKey:   elem.sourceGroupKey,
			Caption:          elem.sourceCaption,
			PreserveCaption:  elem.preserveCaption,
			MetadataJSON:     elem.metadataJSON,
			MetadataGroupKey: elem.metadataGroupKey,
		})
	}
	for index, item := range items {
		t.recordDownloadComplete(successElems[index].ID, item.Size)
	}
	if err := t.saveBatchItems(ctx, successElems, items); err != nil {
		return err
	}
	for _, elem := range successElems {
		if err := core.MarkTaskElementDone(ctx, t.ID, elem.ID); err != nil {
			return fmt.Errorf("failed to persist element completion: %w", err)
		}
	}
	cleanup = successElems
	return nil
}

func (t *Task) saveBatchItems(ctx context.Context, successElems []*TaskElement, items []storagetypes.BatchItem) error {
	t.startUpload(ctx)
	if progressSaver, ok := successElems[0].Storage.(storage.StorageBatchProgressSaver); ok {
		err := progressSaver.SaveBatchWithProgress(ctx, items, func(index int, uploaded, total int64) {
			if index < 0 || index >= len(successElems) {
				return
			}
			t.uploadCallback(ctx, successElems[index].ID)(uploaded, total)
		})
		if err != nil {
			for _, elem := range successElems {
				t.markItemFailed(ctx, elem.ID, FailureStageBatchUpload, err)
			}
			t.notifyStateChange(ctx)
			return fmt.Errorf("failed to save batch: %w", err)
		}
		for index, elem := range successElems {
			t.uploadCallback(ctx, elem.ID)(items[index].Size, items[index].Size)
			t.markItemCompleted(elem.ID)
		}
		t.notifyStateChange(ctx)
		return nil
	}
	for i := range items {
		items[i].Reader = ioutil.NewProgressReader(
			items[i].Reader,
			items[i].Size,
			t.uploadCallback(ctx, successElems[i].ID),
		)
	}
	if err := successElems[0].Storage.(storage.StorageBatchSaver).SaveBatch(ctx, items); err != nil {
		for _, elem := range successElems {
			t.markItemFailed(ctx, elem.ID, FailureStageBatchUpload, err)
		}
		t.notifyStateChange(ctx)
		return fmt.Errorf("failed to save batch: %w", err)
	}
	for index, elem := range successElems {
		t.uploadCallback(ctx, elem.ID)(items[index].Size, items[index].Size)
		t.markItemCompleted(elem.ID)
	}
	t.notifyStateChange(ctx)
	return nil
}

func (t *Task) markProcessing(ctx context.Context, elem *TaskElement) error {
	t.processingMu.Lock()
	if t.processing[elem.ID] != nil {
		t.processingMu.Unlock()
		return fmt.Errorf("element with ID %s is already being processed", elem.ID)
	}
	t.processing[elem.ID] = elem
	t.processingMu.Unlock()
	t.markItemActive(elem.ID, elem.stream, time.Now())
	t.notifyProgress(ctx)
	return nil
}

func (t *Task) unmarkProcessing(id string) {
	t.processingMu.Lock()
	delete(t.processing, id)
	t.processingMu.Unlock()
}

func (t *Task) downloadElement(ctx context.Context, elem *TaskElement) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", elem.File.Name()))
	logger.Info("Starting file download")
	if err := tdler.DownloadToCache(ctx, elem.File, elem.localPath, func(w io.WriterAt) io.WriterAt {
		return ioutil.NewProgressWriterAt(w, func(n int) {
			t.recordItemDownload(elem.ID, int64(n), time.Now())
			downloaded := t.downloaded.Add(int64(n))
			t.notifyProgress(ctx)
			taskevent.Emit(ctx, taskevent.Event{
				TaskID:          t.ID,
				Phase:           taskevent.PhaseProgress,
				TotalBytes:      t.totalSize,
				DownloadedBytes: downloaded,
			})
		})
	}); err != nil {
		t.markItemFailed(ctx, elem.ID, FailureStageDownload, err)
		t.notifyStateChange(ctx)
		return fmt.Errorf("failed to download file: %w", err)
	}
	logger.Info("File downloaded successfully")
	fileStat, err := os.Stat(elem.localPath)
	if err != nil {
		t.markItemFailed(ctx, elem.ID, FailureStageCache, err)
		t.notifyStateChange(ctx)
		return fmt.Errorf("failed to get file stat: %w", err)
	}
	if elem.File.Size() > 0 && fileStat.Size() == 0 {
		err := fmt.Errorf("downloaded file is empty (expected %d bytes)", elem.File.Size())
		t.markItemFailed(ctx, elem.ID, FailureStageDownload, err)
		t.notifyStateChange(ctx)
		return err
	}
	if path.Ext(elem.FileName()) == "" && path.Ext(elem.Path) == "" {
		if ext := fsutil.DetectFileExt(elem.localPath); ext != "" {
			elem.Path += ext
		}
	}
	t.markItemDownloaded(elem.ID)
	t.notifyProgress(ctx)
	return nil
}

func (t *Task) processElement(ctx context.Context, elem *TaskElement) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", elem.File.Name()))
	if elem.stream {
		pr, pw := io.Pipe()
		defer pr.Close()
		errg, uploadCtx := errgroup.WithContext(ctx)
		uploadCtx = storagetypes.WithMessageMetadata(uploadCtx, elem.metadataJSON)
		if elem.preserveCaption {
			uploadCtx = storagetypes.WithSourceCaption(uploadCtx, elem.sourceCaption)
		}
		savedPath := elem.Path
		errg.Go(func() error {
			var err error
			savedPath, err = storage.SaveWithPath(uploadCtx, elem.Storage, pr, elem.Path)
			if err != nil {
				t.markItemFailed(uploadCtx, elem.ID, FailureStageUpload, err)
				t.notifyStateChange(ctx)
			}
			return err
		})
		wr := ioutil.NewProgressWriter(pw, func(n int) {
			t.recordItemDownload(elem.ID, int64(n), time.Now())
			downloaded := t.downloaded.Add(int64(n))
			t.notifyProgress(ctx)
			taskevent.Emit(ctx, taskevent.Event{
				TaskID:          t.ID,
				Phase:           taskevent.PhaseProgress,
				TotalBytes:      t.totalSize,
				DownloadedBytes: downloaded,
			})
		})
		errg.Go(func() error {
			defer pw.Close()
			logger.Info("Starting file download in stream mode")
			downloader, cleanup := tdler.NewDownloader(uploadCtx, elem.File)
			if cleanup != nil {
				defer cleanup(uploadCtx)
			}
			_, err := downloader.Stream(uploadCtx, wr)
			if err != nil {
				logger.Errorf("Failed to download file: %v", err)
				t.markItemFailed(uploadCtx, elem.ID, FailureStageDownload, err)
				t.notifyStateChange(ctx)
				pw.CloseWithError(err)
			}
			return err
		})
		if err := errg.Wait(); err != nil {
			return fmt.Errorf("failed to download file in stream mode: %w", err)
		}
		// Streamed bytes are the uploaded bytes.
		var streamedBytes int64
		t.updateItem(elem.ID, func(item *itemProgressState) {
			streamedBytes = item.downloaded
		})
		t.recordDownloadComplete(elem.ID, streamedBytes)
		t.markItemCompleted(elem.ID)
		if err := core.MarkTaskElementDone(ctx, t.ID, elem.ID); err != nil {
			t.markItemFailed(ctx, elem.ID, FailureStageUpload, err)
			t.notifyStateChange(ctx)
			return fmt.Errorf("failed to persist element completion: %w", err)
		}
		elem.saveMetadataSidecar(ctx, savedPath)
		t.notifyStateChange(ctx)
		logger.Info("File downloaded successfully in stream mode")
		return nil
	}
	logger.Info("Starting file download")
	if err := tdler.DownloadToCache(ctx, elem.File, elem.localPath, func(w io.WriterAt) io.WriterAt {
		return ioutil.NewProgressWriterAt(w, func(n int) {
			t.recordItemDownload(elem.ID, int64(n), time.Now())
			downloaded := t.downloaded.Add(int64(n))
			t.notifyProgress(ctx)
			taskevent.Emit(ctx, taskevent.Event{
				TaskID:          t.ID,
				Phase:           taskevent.PhaseProgress,
				TotalBytes:      t.totalSize,
				DownloadedBytes: downloaded,
			})
		})
	}); err != nil {
		t.markItemFailed(ctx, elem.ID, FailureStageDownload, err)
		t.notifyStateChange(ctx)
		return fmt.Errorf("failed to download file: %w", err)
	}
	logger.Info("File downloaded successfully")
	if path.Ext(elem.FileName()) == "" {
		ext := fsutil.DetectFileExt(elem.localPath)
		if ext != "" {
			elem.Path = elem.Path + ext
		}
	}
	fileStat, err := os.Stat(elem.localPath)
	if err != nil {
		t.markItemFailed(ctx, elem.ID, FailureStageCache, err)
		t.notifyStateChange(ctx)
		return fmt.Errorf("failed to get file stat: %w", err)
	}
	if elem.File.Size() > 0 && fileStat.Size() == 0 {
		err := fmt.Errorf("downloaded file is empty (expected %d bytes)", elem.File.Size())
		t.markItemFailed(ctx, elem.ID, FailureStageDownload, err)
		t.notifyStateChange(ctx)
		return err
	}
	t.recordDownloadComplete(elem.ID, fileStat.Size())
	vctx := context.WithValue(ctx, ctxkey.ContentLength, fileStat.Size())
	vctx = storagetypes.WithMessageMetadata(vctx, elem.metadataJSON)
	if elem.preserveCaption {
		vctx = storagetypes.WithSourceCaption(vctx, elem.sourceCaption)
	}
	t.startUpload(vctx)
	onProgress := t.uploadCallback(vctx, elem.ID)
	attempt := 0
	retryLimit := int(config.C().Retry)
	lastFailureStage := FailureStageUpload
	savedPath := elem.Path
	err = retry.Retry(func() error {
		attempt++
		var file *os.File
		file, err = os.Open(elem.localPath)
		if err != nil {
			lastFailureStage = FailureStageCache
			t.markItemRetry(elem.ID, lastFailureStage, attempt, retryLimit, err)
			t.notifyStateChange(vctx)
			return fmt.Errorf("failed to open cache file: %w", err)
		}
		defer file.Close()
		onProgress(0, fileStat.Size())
		if progressSaver, ok := elem.Storage.(storage.StorageProgressSaver); ok {
			err = progressSaver.SaveWithProgress(vctx, file, elem.Path, onProgress)
			savedPath = elem.Path
		} else {
			savedPath, err = storage.SaveWithPath(vctx, elem.Storage, ioutil.NewProgressReader(file, fileStat.Size(), onProgress), elem.Path)
		}
		if err != nil {
			logger.Errorf("Failed to save file: %s, retrying...", err)
			lastFailureStage = t.itemFailureStage(elem.ID)
			t.markItemRetry(elem.ID, lastFailureStage, attempt, retryLimit, err)
			t.notifyStateChange(vctx)
			return err
		}
		return nil
	}, retry.Context(vctx), retry.RetryTimes(uint(config.C().Retry)))
	if err == nil {
		onProgress(fileStat.Size(), fileStat.Size())
		t.markItemCompleted(elem.ID)
		if err := core.MarkTaskElementDone(vctx, t.ID, elem.ID); err != nil {
			t.markItemFailed(vctx, elem.ID, FailureStageUpload, err)
			t.notifyStateChange(vctx)
			return fmt.Errorf("failed to persist element completion: %w", err)
		}
		elem.saveMetadataSidecar(vctx, savedPath)
		t.notifyStateChange(vctx)
		if err := os.Remove(elem.localPath); err != nil {
			logger.Errorf("Failed to remove cache file: %v", err)
		}
	} else {
		t.markItemFailed(vctx, elem.ID, lastFailureStage, err)
		t.notifyStateChange(vctx)
	}
	return err
}
