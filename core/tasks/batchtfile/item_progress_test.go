package batchtfile

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

func TestTransferMeterTracksHighFrequencyCallbacks(t *testing.T) {
	var meter transferMeter
	started := time.Unix(100, 0)
	meter.record(started, 0)
	for index := 1; index <= 10; index++ {
		meter.record(started.Add(time.Duration(index)*50*time.Millisecond), int64(index*100))
	}

	if got := meter.speed(); got != 2000 {
		t.Fatalf("speed = %.2f B/s, want 2000 B/s", got)
	}
}

func TestTransferMeterDropsStaleSamplesAndResetsOnRetry(t *testing.T) {
	var meter transferMeter
	started := time.Unix(100, 0)
	meter.record(started, 0)
	meter.record(started.Add(time.Second), 100)
	meter.record(started.Add(6*time.Second), 600)
	if got := meter.speed(); got != 100 {
		t.Fatalf("windowed speed = %.2f B/s, want 100 B/s", got)
	}

	meter.record(started.Add(7*time.Second), 10)
	if got := meter.speed(); got != 0 {
		t.Fatalf("speed after retry reset = %.2f B/s, want 0 B/s", got)
	}
}

func TestTransferMeterHandlesOutOfOrderCallbackTimes(t *testing.T) {
	var meter transferMeter
	started := time.Unix(100, 0)
	meter.record(started, 0)
	meter.record(started.Add(time.Second), 100)
	meter.record(started.Add(500*time.Millisecond), 200)

	if got := meter.speed(); got != 200 {
		t.Fatalf("speed after out-of-order callback = %.2f B/s, want 200 B/s", got)
	}
}

func TestStreamingItemTracksBothDirections(t *testing.T) {
	task := progressTestTask(progressTestFile{"stream", 1000})
	started := time.Unix(100, 0)
	task.markItemActive("stream", true, started)
	task.recordItemDownload("stream", 500, started.Add(time.Second))

	item := task.Items()[0]
	if item.Phase != ItemPhaseTransferring {
		t.Fatalf("phase = %v, want transferring", item.Phase)
	}
	if item.Downloaded != 500 || item.Uploaded != 500 {
		t.Fatalf("streamed bytes = download %d, upload %d; want 500/500", item.Downloaded, item.Uploaded)
	}
	if item.DownloadSpeed != 500 || item.UploadSpeed != 500 {
		t.Fatalf("streamed speeds = download %.2f, upload %.2f; want 500/500", item.DownloadSpeed, item.UploadSpeed)
	}
}

func TestDownloadCompletionWaitsForFirstUploadProgress(t *testing.T) {
	task := progressTestTask(progressTestFile{"file", 1000})
	started := time.Unix(100, 0)
	task.markItemActive("file", false, started)
	task.recordItemDownload("file", 1000, started.Add(time.Second))
	task.recordDownloadComplete("file", 1000)

	item := task.Items()[0]
	if item.Phase != ItemPhaseDownloaded {
		t.Fatalf("phase after download completion = %v, want downloaded", item.Phase)
	}
	task.recordItemUpload("file", 1, 1000, started.Add(2*time.Second))
	if got := task.Items()[0].Phase; got != ItemPhaseUploading {
		t.Fatalf("phase after first upload callback = %v, want uploading", got)
	}
}

func TestItemUploadIgnoresOutOfOrderBytesAndAllowsExplicitRetryReset(t *testing.T) {
	task := progressTestTask(progressTestFile{"file", 1000})
	started := time.Unix(100, 0)
	task.recordDownloadComplete("file", 1000)
	task.recordItemUpload("file", 800, 1000, started)
	task.recordItemUpload("file", 400, 1000, started.Add(time.Second))

	item := task.Items()[0]
	if item.Uploaded != 800 || item.Phase != ItemPhaseUploading {
		t.Fatalf("out-of-order callback changed item to %d/%v, want 800/uploading", item.Uploaded, item.Phase)
	}

	task.markItemRetry("file", FailureStageUpload, 1, 3, errors.New("retry"))
	task.recordItemUpload("file", 0, 1000, started.Add(2*time.Second))
	item = task.Items()[0]
	if item.Uploaded != 0 || item.Phase != ItemPhaseUploading {
		t.Fatalf("explicit retry reset item to %d/%v, want 0/uploading", item.Uploaded, item.Phase)
	}
}

func TestItemProgressSnapshotsAreConcurrentSafe(t *testing.T) {
	task := progressTestTask(progressTestFile{"first", 1000})
	task.markItemActive("first", false, time.Unix(100, 0))

	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		for index := 0; index < 1000; index++ {
			task.recordItemDownload("first", 1, time.Unix(100, int64(index+1)))
		}
	}()
	go func() {
		defer wait.Done()
		for index := 0; index < 1000; index++ {
			_ = task.Items()
		}
	}()
	wait.Wait()

	if got := task.Items()[0].Downloaded; got != 1000 {
		t.Fatalf("downloaded = %d, want 1000", got)
	}
}

type progressTestFile struct {
	id   string
	size int64
}

func progressTestTask(files ...progressTestFile) *Task {
	elems := make([]TaskElement, 0, len(files))
	for _, file := range files {
		elems = append(elems, TaskElement{
			ID:   file.id,
			File: tfile.NewTGFile(nil, nil, file.size, file.id+".bin"),
		})
	}
	return NewBatchTGFileTask("progress-test", context.Background(), elems, nil, true)
}
