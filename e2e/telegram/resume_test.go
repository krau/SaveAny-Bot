//go:build e2e

package telegrame2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/config"
	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/core/resume"
	"github.com/krau/SaveAny-Bot/core/tasks/batchtfile"
	"github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
	tfilepkg "github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/rs/xid"
)

// failingDownloadClient fails after failAfter successful chunk requests and
// records the offsets requested by the downloader.
type failingDownloadClient struct {
	downloader.Client
	failAfter int
	mu        sync.Mutex
	calls     int
	offsets   []int64
}

func (c *failingDownloadClient) UploadGetFile(ctx context.Context, req *tg.UploadGetFileRequest) (tg.UploadFileClass, error) {
	c.mu.Lock()
	c.calls++
	c.offsets = append(c.offsets, req.Offset)
	fail := c.failAfter > 0 && c.calls > c.failAfter
	c.mu.Unlock()
	if fail {
		return nil, errors.New("mock download failure")
	}
	return c.Client.UploadGetFile(ctx, req)
}

func (c *failingDownloadClient) requested() []int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]int64(nil), c.offsets...)
}

// recordingStorage captures the bytes uploaded by a task.
type recordingStorage struct {
	mu    sync.Mutex
	data  []byte
	files map[string][]byte
}

func (s *recordingStorage) Init(context.Context, storcfg.StorageConfig) error { return nil }
func (s *recordingStorage) Type() storenum.StorageType                        { return storenum.Local }
func (s *recordingStorage) Name() string                                      { return "mock" }
func (s *recordingStorage) Save(_ context.Context, r io.Reader, path string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.data = data
	if s.files == nil {
		s.files = make(map[string][]byte)
	}
	s.files[path] = data
	s.mu.Unlock()
	return nil
}
func (s *recordingStorage) Exists(context.Context, string) (bool, error) { return false, nil }

func (s *recordingStorage) saved(path string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.files[path]...)
}

func TestTelegramTaskDownloadResumesAfterMockFailure(t *testing.T) {
	harness.Server.ResetRecords()
	content := bytes.Repeat([]byte("resume-data-"), (12*1024*1024)/12+1)[:12*1024*1024]
	const name = "resume.bin"
	document := harness.Server.AddDocumentMessage(
		harness.UserPeer(localUserID),
		harness.UserPeer(localUserID),
		name,
		content,
	)
	media, ok := document.Media.(*tg.MessageMediaDocument)
	if !ok {
		t.Fatalf("message media = %T", document.Media)
	}
	doc, ok := media.Document.AsNotEmpty()
	if !ok {
		t.Fatal("document is empty")
	}

	taskID := "resume-" + xid.New().String()
	params, err := json.Marshal(taskspec.TGFile{Items: []taskspec.TGFileItem{{
		ID: taskID, ChatID: localUserID, MessageID: document.ID, Name: name,
		Storage: "mock", Path: name,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	record := &database.TaskRecord{
		TaskID: taskID, Type: tasktype.TaskTypeTgfiles.String(), Status: "running", Params: string(params),
	}
	if err := database.CreateTaskRecord(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	defer database.DeleteTaskRecord(context.Background(), taskID)

	base := tg.NewClient(harness.Server)
	location := &tg.InputDocumentFileLocation{
		ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference,
	}
	firstClient := &failingDownloadClient{Client: base, failAfter: 3}
	file := tfilepkg.NewTGFile(location, firstClient, int64(len(content)), name)
	storage := &recordingStorage{}
	first, err := tfile.NewTGFileTask(taskID, context.Background(), file, storage, name, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Execute(context.Background()); err == nil {
		t.Fatal("first download unexpectedly succeeded")
	}
	t.Logf("first run offsets: %v", firstClient.requested())
	if len(firstClient.requested()) < 3 {
		t.Fatalf("first download requested %d chunks, want at least 3", len(firstClient.requested()))
	}

	cachePath, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, fmt.Sprintf("%s_%s", taskID, name)))
	if err != nil {
		t.Fatal(err)
	}
	partPath := cachePath + ".part"
	bitmapPath := partPath + ".bitmap"
	if _, err := os.Stat(partPath); err != nil {
		t.Fatalf("partial cache not retained: %v", err)
	}
	if _, err := os.Stat(bitmapPath); err != nil {
		t.Fatalf("resume bitmap not retained: %v", err)
	}

	secondClient := &failingDownloadClient{Client: base}
	file = tfilepkg.NewTGFile(location, secondClient, int64(len(content)), name)
	second, err := tfile.NewTGFileTask(taskID, context.Background(), file, storage, name, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Execute(context.Background()); err != nil {
		t.Fatalf("resumed download failed: %v", err)
	}
	if !bytes.Equal(storage.data, content) {
		t.Fatalf("uploaded content size = %d, want %d", len(storage.data), len(content))
	}
	t.Logf("resume offsets: %v", secondClient.requested())
	for _, offset := range secondClient.requested() {
		if offset < 2*1024*1024 {
			t.Fatalf("resume re-requested completed offset %d", offset)
		}
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("cache still exists after upload: %v", err)
	}
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Fatalf("part file still exists after upload: %v", err)
	}
	if _, err := os.Stat(bitmapPath); !os.IsNotExist(err) {
		t.Fatalf("bitmap still exists after upload: %v", err)
	}
	saved, err := database.GetTaskRecord(context.Background(), taskID)
	if err != nil {
		t.Fatal(err)
	}
	var savedSpec taskspec.TGFile
	if err := json.Unmarshal([]byte(saved.Params), &savedSpec); err != nil {
		t.Fatalf("invalid persisted params: %v", err)
	}
	if !slices.Contains(savedSpec.Done, taskID) {
		t.Fatalf("persisted done list = %v, want %s", savedSpec.Done, taskID)
	}

	// A queued record whose done list is complete must be finalized without a
	// Telegram client or another download.
	if err := database.UpdateTaskRecord(context.Background(), taskID, map[string]any{"status": "queued"}); err != nil {
		t.Fatal(err)
	}
	resume.Run(context.Background())
	recovered, err := database.GetTaskRecord(context.Background(), taskID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status != "completed" {
		t.Fatalf("recovered status = %q, want completed", recovered.Status)
	}
}

func TestTelegramBatchDownloadResumesAfterMockFailure(t *testing.T) {
	harness.Server.ResetRecords()
	type fileCase struct {
		name    string
		content []byte
		id      string
		doc     *tg.Document
	}
	files := []fileCase{
		{name: "batch-a.bin", content: bytes.Repeat([]byte("a"), 4*1024*1024), id: xid.New().String()},
		{name: "batch-b.bin", content: bytes.Repeat([]byte("b"), 4*1024*1024), id: xid.New().String()},
	}
	base := tg.NewClient(harness.Server)
	storage := &recordingStorage{}
	taskID := "batch-resume-" + xid.New().String()
	items := make([]taskspec.TGFileItem, 0, len(files))
	for i := range files {
		document := harness.Server.AddDocumentMessage(
			harness.UserPeer(localUserID), harness.UserPeer(localUserID), files[i].name, files[i].content,
		)
		media := document.Media.(*tg.MessageMediaDocument)
		doc, ok := media.Document.AsNotEmpty()
		if !ok {
			t.Fatal("empty document")
		}
		files[i].doc = doc
		items = append(items, taskspec.TGFileItem{
			ID: files[i].id, ChatID: localUserID, MessageID: document.ID, Name: files[i].name, Storage: "mock", Path: files[i].name,
		})
	}
	params, _ := json.Marshal(taskspec.TGFile{Items: items})
	record := &database.TaskRecord{TaskID: taskID, Type: tasktype.TaskTypeTgfiles.String(), Status: "running", Params: string(params)}
	if err := database.CreateTaskRecord(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	defer database.DeleteTaskRecord(context.Background(), taskID)

	makeElems := func(failAfter int) []batchtfile.TaskElement {
		elems := make([]batchtfile.TaskElement, 0, len(files))
		for _, tc := range files {
			location := &tg.InputDocumentFileLocation{ID: tc.doc.ID, AccessHash: tc.doc.AccessHash, FileReference: tc.doc.FileReference}
			client := &failingDownloadClient{Client: base, failAfter: failAfter}
			file := tfilepkg.NewTGFile(location, client, int64(len(tc.content)), tc.name)
			elem, err := batchtfile.NewTaskElementWithID(tc.id, storage, tc.name, file)
			if err != nil {
				t.Fatal(err)
			}
			elems = append(elems, *elem)
		}
		return elems
	}

	first := batchtfile.NewBatchTGFileTask(taskID, context.Background(), makeElems(1), nil, false)
	if err := first.Execute(context.Background()); err == nil {
		t.Fatal("first batch download unexpectedly succeeded")
	}
	for _, tc := range files {
		part, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, tc.id+"_"+tc.name+".part"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(part); err != nil {
			t.Fatalf("partial batch cache missing for %s: %v", tc.name, err)
		}
	}

	second := batchtfile.NewBatchTGFileTask(taskID, context.Background(), makeElems(0), nil, false)
	if err := second.Execute(context.Background()); err != nil {
		t.Fatalf("resumed batch failed: %v", err)
	}
	for _, tc := range files {
		if got := storage.saved(tc.name); !bytes.Equal(got, tc.content) {
			t.Fatalf("uploaded %s size=%d, want %d", tc.name, len(got), len(tc.content))
		}
		cache, err := filepath.Abs(filepath.Join(config.C().Temp.BasePath, tc.id+"_"+tc.name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(cache); !os.IsNotExist(err) {
			t.Fatalf("completed batch cache still exists for %s: %v", tc.name, err)
		}
	}
}
