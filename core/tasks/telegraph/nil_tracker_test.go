package telegraph_test

import (
	"context"
	"io"
	"testing"

	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/core/tasks/telegraph"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/storage"
)

type mockStorage struct{}

func (mockStorage) Init(context.Context, storconfig.StorageConfig) error { return nil }
func (mockStorage) Type() storenum.StorageType                           { return storenum.Local }
func (mockStorage) Name() string                                         { return "mock" }
func (mockStorage) Save(context.Context, io.Reader, string) error        { return nil }
func (mockStorage) Exists(context.Context, string) bool                  { return false }

var _ storage.Storage = mockStorage{}

// Regression: API-created tasks run with a nil ProgressTracker (progress goes
// through taskevent); Execute must not panic on the tracker callbacks.
func TestExecuteWithNilTracker(t *testing.T) {
	task := telegraph.NewTask("id", t.Context(), "/page", nil, mockStorage{}, "/out", nil, nil)
	if err := task.Execute(t.Context()); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}
