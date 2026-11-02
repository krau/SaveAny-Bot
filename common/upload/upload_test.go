package upload

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

func TestSaveRetriesAndRewinds(t *testing.T) {
	stor := &retryStorage{failures: 1}
	err := Save(context.Background(), strings.NewReader("payload"), 7, "file.txt", Options{
		Storage:    stor,
		Retries:    2,
		RetryDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if stor.attempts != 2 {
		t.Fatalf("attempts = %d, want 2", stor.attempts)
	}
	if string(stor.data) != "payload" {
		t.Fatalf("uploaded data = %q, want payload", stor.data)
	}
}

type retryStorage struct {
	attempts int
	failures int
	data     []byte
}

func (s *retryStorage) Init(context.Context, storage.StorageConfig) error { return nil }
func (s *retryStorage) Type() storenum.StorageType                        { return storenum.Local }
func (s *retryStorage) Name() string                                      { return "test" }
func (s *retryStorage) Exists(context.Context, string) (bool, error)      { return false, nil }

func (s *retryStorage) Save(_ context.Context, r io.Reader, _ string) error {
	s.attempts++
	if s.attempts <= s.failures {
		return errors.New("transient failure")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.data = data
	return nil
}
