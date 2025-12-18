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

	if !s.Exists(ctx, key) {
		t.Fatalf("Exists should return true for saved key")
	}

	if s.Exists(ctx, "nonexistent.txt") {
		t.Fatalf("Exists should return false for nonexistent key")
	}

	if err := s.Save(ctx, bytes.NewReader(content), key); err != nil {
		t.Fatalf("Save with existing key failed: %v", err)
	}

	if !s.Exists(ctx, "foo/bar_1.txt") {
		t.Fatalf("Exists should return true for unique renamed key")
	}

	var length int64 = int64(len(content))
	ctx = context.WithValue(ctx, ctxkey.ContentLength, length)
	if err := s.Save(ctx, bytes.NewReader(content), "size_test.txt"); err != nil {
		t.Fatalf("Save with content length failed: %v", err)
	}

	if !s.Exists(ctx, "size_test.txt") {
		t.Fatalf("Exists should return true for size_test.txt")
	}
}
