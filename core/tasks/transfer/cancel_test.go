package transfer_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/krau/SaveAny-Bot/config"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/core/tasks/transfer"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/storage"
)

// initConfig seeds the global config so task execution reads a sane Workers
// value (the zero default would deadlock errgroup.SetLimit(0)).
func initConfig(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("workers = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.Init(t.Context(), path); err != nil {
		t.Fatal(err)
	}
}

type cancelSource struct{}

func (cancelSource) Init(context.Context, storconfig.StorageConfig) error { return nil }
func (cancelSource) Type() storenum.StorageType                           { return storenum.Local }
func (cancelSource) Name() string                                         { return "cancel-source" }
func (cancelSource) Save(context.Context, io.Reader, string) error        { return nil }
func (cancelSource) Exists(context.Context, string) bool                  { return false }
func (cancelSource) ListFiles(context.Context, string) ([]storagetypes.FileInfo, error) {
	return nil, nil
}

// OpenFile reports cancellation, as the task context would be cancelled.
func (cancelSource) OpenFile(ctx context.Context, path string) (io.ReadCloser, int64, error) {
	return nil, 0, ctx.Err()
}

type voidTarget struct{}

func (voidTarget) Init(context.Context, storconfig.StorageConfig) error { return nil }
func (voidTarget) Type() storenum.StorageType                           { return storenum.Local }
func (voidTarget) Name() string                                         { return "void-target" }
func (voidTarget) Save(context.Context, io.Reader, string) error        { return nil }
func (voidTarget) Exists(context.Context, string) bool                  { return false }

var (
	_ storage.StorageReadable = cancelSource{}
	_ storage.Storage         = voidTarget{}
)

// Regression: IgnoreErrors must swallow element failures but never a task
// cancellation; Execute must surface context.Canceled.
func TestIgnoreErrorsPropagatesCancel(t *testing.T) {
	initConfig(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	elem := transfer.NewTaskElement(cancelSource{}, storagetypes.FileInfo{Path: "/a", Name: "a", Size: 1}, voidTarget{}, "/out")
	task := transfer.NewTransferTask("id", ctx, []transfer.TaskElement{*elem}, nil, true)

	err := task.Execute(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
