package batchtfile

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
	t.Progress.OnStart(ctx, t)
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
			break
		}
	}
	if err != nil {
		logger.Errorf("Error during batch file processing: %v", err)
	} else {
		logger.Info("Batch file task completed successfully")
	}
	t.Progress.OnDone(ctx, t, err)
	return err
}

func (t *Task) executionGroups() []executionGroup {
	groups := make([]executionGroup, 0, len(t.elems))
	for i := 0; i < len(t.elems); {
		elem := &t.elems[i]
		batchSaver, batchCapable := elem.Storage.(storage.StorageBatchSaver)
		if !batchCapable || elem.sourceGroupKey == "" {
			groups = append(groups, executionGroup{elems: []*TaskElement{elem}})
			i++
			continue
		}

		end := i + 1
		for end < len(t.elems) {
			next := &t.elems[end]
			if next.Storage != elem.Storage || next.sourceGroupKey != elem.sourceGroupKey {
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
			if err := t.markProcessing(elem); err != nil {
				return err
			}
			defer t.unmarkProcessing(elem.ID)
			return t.processElement(gctx, *elem)
		})
	}
	return eg.Wait()
}

func (t *Task) processBatch(ctx context.Context, group executionGroup) error {
	defer func() {
		for _, elem := range group.elems {
			if err := os.Remove(elem.localPath); err != nil && !os.IsNotExist(err) {
				log.FromContext(ctx).Warnf("Failed to cleanup batch cache file %s: %v", elem.localPath, err)
			}
		}
	}()

	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(config.C().Workers)
	for _, elem := range group.elems {
		eg.Go(func() error {
			if err := t.markProcessing(elem); err != nil {
				return err
			}
			defer t.unmarkProcessing(elem.ID)
			return t.downloadElement(gctx, elem)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	items := make([]storagetypes.BatchItem, 0, len(group.elems))
	openFiles := make([]*os.File, 0, len(group.elems))
	defer func() {
		for _, file := range openFiles {
			if err := file.Close(); err != nil {
				log.FromContext(ctx).Warnf("Failed to close batch cache file %s: %v", file.Name(), err)
			}
		}
	}()
	for _, elem := range group.elems {
		file, err := os.Open(elem.localPath)
		if err != nil {
			return fmt.Errorf("failed to open cache file: %w", err)
		}
		stat, err := file.Stat()
		if err != nil {
			file.Close()
			return fmt.Errorf("failed to get cache file stat: %w", err)
		}
		openFiles = append(openFiles, file)
		items = append(items, storagetypes.BatchItem{
			Reader:          file,
			StoragePath:     elem.Path,
			Size:            stat.Size(),
			SourceGroupKey:  elem.sourceGroupKey,
			Caption:         elem.sourceCaption,
			PreserveCaption: elem.preserveCaption,
		})
	}
	return t.saveBatchItems(ctx, group, items)
}

func (t *Task) saveBatchItems(ctx context.Context, group executionGroup, items []storagetypes.BatchItem) error {
	t.startUpload(ctx)
	if progressSaver, ok := group.batchSaver.(storage.StorageBatchProgressSaver); ok {
		err := progressSaver.SaveBatchWithProgress(ctx, items, func(index int, uploaded, total int64) {
			if index < 0 || index >= len(group.elems) {
				return
			}
			t.uploadCallback(ctx, group.elems[index].ID)(uploaded, total)
		})
		if err != nil {
			return fmt.Errorf("failed to save batch: %w", err)
		}
		for index, elem := range group.elems {
			t.uploadCallback(ctx, elem.ID)(items[index].Size, items[index].Size)
		}
		return nil
	}
	for i := range items {
		items[i].Reader = ioutil.NewProgressReader(
			items[i].Reader,
			items[i].Size,
			t.uploadCallback(ctx, group.elems[i].ID),
		)
	}
	if err := group.batchSaver.SaveBatch(ctx, items); err != nil {
		return fmt.Errorf("failed to save batch: %w", err)
	}
	for index, elem := range group.elems {
		t.uploadCallback(ctx, elem.ID)(items[index].Size, items[index].Size)
	}
	return nil
}

func (t *Task) markProcessing(elem *TaskElement) error {
	t.processingMu.Lock()
	defer t.processingMu.Unlock()
	if t.processing[elem.ID] != nil {
		return fmt.Errorf("element with ID %s is already being processed", elem.ID)
	}
	t.processing[elem.ID] = elem
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
	localFile, err := fsutil.CreateFile(elem.localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	wrAt := ioutil.NewProgressWriterAt(localFile, func(n int) {
		downloaded := t.downloaded.Add(int64(n))
		t.Progress.OnProgress(ctx, t)
		taskevent.Emit(ctx, taskevent.Event{
			TaskID:          t.ID,
			Phase:           taskevent.PhaseProgress,
			TotalBytes:      t.totalSize,
			DownloadedBytes: downloaded,
		})
	})
	_, downloadErr := tdler.NewDownloader(elem.File).Parallel(ctx, wrAt)
	closeErr := localFile.Close()
	if downloadErr != nil {
		return fmt.Errorf("failed to download file: %w", downloadErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close cache file: %w", closeErr)
	}
	logger.Info("File downloaded successfully")
	if path.Ext(elem.FileName()) == "" {
		if ext := fsutil.DetectFileExt(elem.localPath); ext != "" {
			elem.Path += ext
		}
	}
	return nil
}

func (t *Task) processElement(ctx context.Context, elem TaskElement) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("file[%s]", elem.File.Name()))
	if elem.stream {
		pr, pw := io.Pipe()
		defer pr.Close()
		errg, uploadCtx := errgroup.WithContext(ctx)
		errg.Go(func() error {
			return elem.Storage.Save(uploadCtx, pr, elem.Path)
		})
		wr := ioutil.NewProgressWriter(pw, func(n int) {
			downloaded := t.downloaded.Add(int64(n))
			t.Progress.OnProgress(ctx, t)
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
			_, err := tdler.NewDownloader(elem.File).Stream(uploadCtx, wr)
			if err != nil {
				logger.Errorf("Failed to download file: %v", err)
				pw.CloseWithError(err)
			}
			return err
		})
		if err := errg.Wait(); err != nil {
			return fmt.Errorf("failed to download file in stream mode: %w", err)
		}
		logger.Info("File downloaded successfully in stream mode")
		return nil
	}
	logger.Info("Starting file download")
	localFile, err := fsutil.CreateFile(elem.localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer func() {
		if err := localFile.CloseAndRemove(); err != nil {
			logger.Errorf("Failed to close local file: %v", err)
		}
	}()
	wrAt := ioutil.NewProgressWriterAt(localFile, func(n int) {
		downloaded := t.downloaded.Add(int64(n))
		t.Progress.OnProgress(ctx, t)
		taskevent.Emit(ctx, taskevent.Event{
			TaskID:          t.ID,
			Phase:           taskevent.PhaseProgress,
			TotalBytes:      t.totalSize,
			DownloadedBytes: downloaded,
		})
	})
	_, err = tdler.NewDownloader(elem.File).Parallel(ctx, wrAt)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	logger.Info("File downloaded successfully")
	if path.Ext(elem.FileName()) == "" {
		ext := fsutil.DetectFileExt(elem.localPath)
		if ext != "" {
			elem.Path = elem.Path + ext
		}
	}
	var fileStat os.FileInfo
	fileStat, err = os.Stat(elem.localPath)
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}
	vctx := context.WithValue(ctx, ctxkey.ContentLength, fileStat.Size())
	t.startUpload(vctx)
	onProgress := t.uploadCallback(vctx, elem.ID)
	err = retry.Retry(func() error {
		var file *os.File
		file, err = os.Open(elem.localPath)
		if err != nil {
			return fmt.Errorf("failed to open cache file: %w", err)
		}
		defer file.Close()
		onProgress(0, fileStat.Size())
		if progressSaver, ok := elem.Storage.(storage.StorageProgressSaver); ok {
			err = progressSaver.SaveWithProgress(vctx, file, elem.Path, onProgress)
		} else {
			err = elem.Storage.Save(vctx, ioutil.NewProgressReader(file, fileStat.Size(), onProgress), elem.Path)
		}
		if err != nil {
			logger.Errorf("Failed to save file: %s, retrying...", err)
			return err
		}
		return nil
	}, retry.Context(vctx), retry.RetryTimes(uint(config.C().Retry)))
	if err == nil {
		onProgress(fileStat.Size(), fileStat.Size())
	}
	return err
}
