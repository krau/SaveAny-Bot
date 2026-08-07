package batchtfile

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	transferSpeedWindow  = 5 * time.Second
	transferSamplePeriod = 250 * time.Millisecond
)

// ItemPhase describes the current lifecycle stage of one batch item.
type ItemPhase uint8

const (
	ItemPhaseWaiting ItemPhase = iota
	ItemPhaseDownloading
	ItemPhaseTransferring
	ItemPhaseDownloaded
	ItemPhaseUploading
	ItemPhaseRetrying
	ItemPhaseConfirming
	ItemPhaseCompleted
	ItemPhaseFailed
	ItemPhaseStopped
)

// FailureStage identifies the operation that failed for one batch item.
type FailureStage uint8

const (
	FailureStageNone FailureStage = iota
	FailureStageDownload
	FailureStageCache
	FailureStageUpload
	FailureStageConfirm
	FailureStageBatchUpload
	FailureStageInternal
)

// TaskItemProgress is an immutable progress snapshot for one batch item.
type TaskItemProgress struct {
	Index         int
	ID            string
	Name          string
	Size          int64
	Downloaded    int64
	Uploaded      int64
	DownloadSpeed float64
	UploadSpeed   float64
	Phase         ItemPhase
	FailureStage  FailureStage
	RetryAttempt  int
	RetryLimit    int
	Error         string
}

type transferSample struct {
	at    time.Time
	bytes int64
}

type transferMeter struct {
	samples []transferSample
	latest  transferSample
	hasData bool
}

func (m *transferMeter) record(now time.Time, transferred int64) {
	if m.hasData && transferred < m.latest.bytes {
		m.reset()
	}
	if m.hasData && now.Before(m.latest.at) {
		now = m.latest.at
	}
	sample := transferSample{at: now, bytes: transferred}
	m.latest = sample
	m.hasData = true
	if len(m.samples) == 0 {
		m.samples = append(m.samples, sample)
		return
	}
	if now.Sub(m.samples[len(m.samples)-1].at) >= transferSamplePeriod {
		m.samples = append(m.samples, sample)
	}
	cutoff := now.Add(-transferSpeedWindow)
	for len(m.samples) > 1 && m.samples[0].at.Before(cutoff) {
		m.samples = m.samples[1:]
	}
}

func (m *transferMeter) speed() float64 {
	if !m.hasData || len(m.samples) == 0 {
		return 0
	}
	first := m.samples[0]
	last := m.latest
	elapsed := last.at.Sub(first.at).Seconds()
	if elapsed <= 0 || last.bytes <= first.bytes {
		return 0
	}
	return float64(last.bytes-first.bytes) / elapsed
}

func (m *transferMeter) reset() {
	m.samples = m.samples[:0]
	m.latest = transferSample{}
	m.hasData = false
}

type itemProgressState struct {
	index         int
	id            string
	name          string
	expectedSize  int64
	actualSize    int64
	downloaded    int64
	uploaded      int64
	phase         ItemPhase
	failureStage  FailureStage
	retryAttempt  int
	retryLimit    int
	err           string
	downloadMeter transferMeter
	uploadMeter   transferMeter
}

func newItemProgressStates(elems []TaskElement) ([]itemProgressState, map[string]int) {
	states := make([]itemProgressState, 0, len(elems))
	index := make(map[string]int, len(elems))
	for i, elem := range elems {
		name := ""
		size := int64(0)
		if elem.File != nil {
			name = elem.File.Name()
			size = elem.File.Size()
		}
		states = append(states, itemProgressState{
			index:        i + 1,
			id:           elem.ID,
			name:         name,
			expectedSize: size,
			phase:        ItemPhaseWaiting,
		})
		index[elem.ID] = i
	}
	return states, index
}

func (t *Task) updateItem(id string, update func(*itemProgressState)) bool {
	t.itemMu.Lock()
	defer t.itemMu.Unlock()
	index, ok := t.itemIndex[id]
	if !ok || index < 0 || index >= len(t.itemStates) {
		return false
	}
	update(&t.itemStates[index])
	return true
}

func (t *Task) markItemActive(id string, stream bool, now time.Time) {
	t.updateItem(id, func(item *itemProgressState) {
		if stream {
			item.phase = ItemPhaseTransferring
		} else {
			item.phase = ItemPhaseDownloading
		}
		item.failureStage = FailureStageNone
		item.err = ""
		item.downloadMeter.record(now, item.downloaded)
		if stream {
			item.uploadMeter.record(now, item.uploaded)
		}
	})
}

func (t *Task) recordItemDownload(id string, n int64, now time.Time) {
	if n <= 0 {
		return
	}
	t.updateItem(id, func(item *itemProgressState) {
		item.downloaded += n
		item.downloadMeter.record(now, item.downloaded)
		if item.phase == ItemPhaseTransferring {
			item.uploaded += n
			item.uploadMeter.record(now, item.uploaded)
		}
	})
}

func (t *Task) markItemDownloaded(id string) {
	t.updateItem(id, func(item *itemProgressState) {
		item.phase = ItemPhaseDownloaded
	})
}

func (t *Task) recordItemDownloaded(id string, actualSize int64) {
	t.updateItem(id, func(item *itemProgressState) {
		if actualSize > 0 {
			item.actualSize = actualSize
		}
		if item.actualSize == 0 {
			item.actualSize = item.downloaded
		}
		if item.phase != ItemPhaseTransferring {
			item.phase = ItemPhaseDownloaded
			item.uploadMeter.reset()
		}
	})
}

func (t *Task) recordItemUpload(id string, uploaded, total int64, now time.Time) bool {
	becameConfirming := false
	t.updateItem(id, func(item *itemProgressState) {
		if uploaded < item.uploaded {
			if item.phase != ItemPhaseRetrying {
				return
			}
			item.uploadMeter.reset()
		}
		if total > 0 {
			item.actualSize = total
		}
		item.uploaded = uploaded
		item.uploadMeter.record(now, uploaded)
		if total > 0 && uploaded >= total {
			becameConfirming = item.phase != ItemPhaseConfirming
			item.phase = ItemPhaseConfirming
			return
		}
		item.phase = ItemPhaseUploading
		item.failureStage = FailureStageNone
		item.err = ""
	})
	return becameConfirming
}

func (t *Task) markItemRetry(id string, stage FailureStage, attempt, limit int, err error) {
	t.updateItem(id, func(item *itemProgressState) {
		item.phase = ItemPhaseRetrying
		item.failureStage = stage
		item.retryAttempt = attempt
		item.retryLimit = limit
		item.err = compactError(err)
	})
}

func (t *Task) markItemFailed(id string, stage FailureStage, err error) {
	t.updateItem(id, func(item *itemProgressState) {
		if item.phase == ItemPhaseFailed || item.phase == ItemPhaseCompleted {
			return
		}
		if errors.Is(err, context.Canceled) {
			item.phase = ItemPhaseStopped
			return
		}
		item.phase = ItemPhaseFailed
		item.failureStage = stage
		item.err = compactError(err)
	})
}

func (t *Task) markItemCompleted(id string) {
	t.updateItem(id, func(item *itemProgressState) {
		item.phase = ItemPhaseCompleted
		item.failureStage = FailureStageNone
		item.err = ""
		item.retryAttempt = 0
		item.retryLimit = 0
		if item.actualSize == 0 {
			item.actualSize = max(item.downloaded, item.uploaded)
		}
	})
}

func (t *Task) finishItems(err error) {
	if err == nil {
		return
	}
	t.itemMu.Lock()
	defer t.itemMu.Unlock()
	for i := range t.itemStates {
		item := &t.itemStates[i]
		if item.phase == ItemPhaseCompleted || item.phase == ItemPhaseFailed {
			continue
		}
		item.phase = ItemPhaseStopped
	}
}

func (t *Task) itemFailureStage(id string) FailureStage {
	t.itemMu.RLock()
	defer t.itemMu.RUnlock()
	index, ok := t.itemIndex[id]
	if !ok || index < 0 || index >= len(t.itemStates) {
		return FailureStageUpload
	}
	if t.itemStates[index].phase == ItemPhaseConfirming {
		return FailureStageConfirm
	}
	return FailureStageUpload
}

func (t *Task) Items() []TaskItemProgress {
	t.itemMu.RLock()
	defer t.itemMu.RUnlock()
	items := make([]TaskItemProgress, 0, len(t.itemStates))
	for i := range t.itemStates {
		item := &t.itemStates[i]
		size := item.actualSize
		if size == 0 {
			size = item.expectedSize
		}
		items = append(items, TaskItemProgress{
			Index:         item.index,
			ID:            item.id,
			Name:          item.name,
			Size:          size,
			Downloaded:    item.downloaded,
			Uploaded:      item.uploaded,
			DownloadSpeed: item.downloadMeter.speed(),
			UploadSpeed:   item.uploadMeter.speed(),
			Phase:         item.phase,
			FailureStage:  item.failureStage,
			RetryAttempt:  item.retryAttempt,
			RetryLimit:    item.retryLimit,
			Error:         item.err,
		})
	}
	return items
}

func (t *Task) ActualTotalSize() int64 {
	items := t.Items()
	var total int64
	for _, item := range items {
		total += item.Size
	}
	return total
}

type stateProgressTracker interface {
	OnStateChange(ctx context.Context, info TaskInfo)
}

func (t *Task) notifyStateChange(ctx context.Context) {
	if tracker, ok := t.Progress.(stateProgressTracker); ok {
		tracker.OnStateChange(ctx, t)
	}
}

func compactError(err error) string {
	if err == nil {
		return ""
	}
	return strings.Join(strings.Fields(err.Error()), " ")
}
