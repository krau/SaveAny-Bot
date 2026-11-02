package s3_test

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/storage/s3"
)

func newTestContext(t *testing.T) context.Context {
	t.Helper()
	logger := log.NewWithOptions(nil, log.Options{ReportTimestamp: false})
	ctx := context.Background()
	return log.WithContext(ctx, logger)
}

func newFakeS3(t *testing.T) (*s3.S3, *storconfig.S3StorageConfig) {
	t.Helper()

	backend := s3mem.New()
	fakeSrv := gofakes3.New(backend)
	ts := httptest.NewServer(fakeSrv.Server())
	t.Cleanup(ts.Close)

	cfg := &storconfig.S3StorageConfig{
		BaseConfig: storconfig.BaseConfig{
			Name:   "test-s3",
			Type:   "s3",
			Enable: true,
		},
		Endpoint:        ts.URL,
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret",
		BucketName:      "test-bucket",
		BasePath:        "base",
		Region:          "us-east-1",
	}

	if err := backend.CreateBucket("test-bucket"); err != nil {
		t.Fatalf("failed to create fake bucket: %v", err)
	}

	s := &s3.S3{}
	ctx := newTestContext(t)
	if err := s.Init(ctx, cfg); err != nil {
		t.Fatalf("init s3 failed: %v", err)
	}

	return s, cfg
}

func TestS3(t *testing.T) {
	s, _ := newFakeS3(t)
	ctx := t.Context()

	content := []byte("hello world")
	reader := bytes.NewReader(content)
	key := "foo/bar.txt"

	if err := s.Save(ctx, reader, key); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if exists, err := s.Exists(ctx, key); err != nil || !exists {
		t.Fatalf("Exists(key) = %v, %v, want true, nil", exists, err)
	}

	if exists, err := s.Exists(ctx, "nonexistent.txt"); err != nil || exists {
		t.Fatalf("Exists(nonexistent) = %v, %v, want false, nil", exists, err)
	}

	if err := s.Save(ctx, bytes.NewReader(content), key); err != nil {
		t.Fatalf("Save with existing key failed: %v", err)
	}

	if exists, err := s.Exists(ctx, "foo/bar_1.txt"); err != nil || !exists {
		t.Fatalf("Exists(renamed) = %v, %v, want true, nil", exists, err)
	}

	var length int64 = int64(len(content))
	ctx = context.WithValue(ctx, ctxkey.ContentLength, length)
	if err := s.Save(ctx, bytes.NewReader(content), "size_test.txt"); err != nil {
		t.Fatalf("Save with content length failed: %v", err)
	}

	if exists, err := s.Exists(ctx, "size_test.txt"); err != nil || !exists {
		t.Fatalf("Exists(size_test) = %v, %v, want true, nil", exists, err)
	}
}

func TestS3MultipartSave(t *testing.T) {
	s, _ := newFakeS3(t)
	ctx := t.Context()
	content := make([]byte, 65<<20)
	for i := range content {
		content[i] = byte(i)
	}
	if err := s.Save(ctx, bytes.NewReader(content), "multipart.bin"); err != nil {
		t.Fatalf("multipart Save failed: %v", err)
	}
	exists, err := s.Exists(ctx, "multipart.bin")
	if err != nil || !exists {
		t.Fatalf("multipart object exists = %v, %v, want true, nil", exists, err)
	}
}
