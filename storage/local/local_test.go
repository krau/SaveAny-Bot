package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krau/SaveAny-Bot/config/storage"
)

func newTestLocal(t *testing.T) *Local {
	t.Helper()
	base := t.TempDir()
	l := &Local{}
	if err := l.Init(context.Background(), &storage.LocalStorageConfig{
		BaseConfig: storage.BaseConfig{Name: "test", Type: "local", Enable: true},
		BasePath:   base,
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return l
}

func TestLocalSaveDoesNotOverwrite(t *testing.T) {
	l := newTestLocal(t)
	ctx := context.Background()
	if err := l.Save(ctx, strings.NewReader("first"), "dir/file.txt"); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if err := l.Save(ctx, strings.NewReader("second"), "dir/file.txt"); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(l.config.BasePath, "dir", "file_1.txt"))
	if err != nil {
		t.Fatalf("read renamed file: %v", err)
	}
	if string(got) != "second" {
		t.Fatalf("renamed file = %q, want second", got)
	}
}

func TestLocalRejectsTraversal(t *testing.T) {
	l := newTestLocal(t)
	err := l.Save(context.Background(), strings.NewReader("x"), "../escape.txt")
	if err == nil {
		t.Fatal("Save traversal returned no error")
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(l.config.BasePath), "escape.txt")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("traversal file was created or stat failed: %v", statErr)
	}
}
