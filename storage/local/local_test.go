package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	configstorage "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
)

func newTestLocal(t *testing.T) *Local {
	t.Helper()
	base := t.TempDir()
	l := &Local{}
	if err := l.Init(context.Background(), &configstorage.LocalStorageConfig{
		Name: "test", Type: "local", Enable: true,
		BasePath: base,
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(l.Close)
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

func TestLocalRejectsSymlinkedParentOutsideRoot(t *testing.T) {
	l := newTestLocal(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(l.config.BasePath, "linked")); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	for _, overwrite := range []bool{false, true} {
		t.Run(map[bool]string{false: "new file", true: "overwrite"}[overwrite], func(t *testing.T) {
			target := filepath.Join(outside, "escape.txt")
			if overwrite {
				if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
					t.Fatalf("write existing outside file: %v", err)
				}
			}

			ctx := context.Background()
			if overwrite {
				ctx = context.WithValue(ctx, ctxkey.OverwriteExisting, true)
			}
			err := l.Save(ctx, strings.NewReader("x"), "linked/escape.txt")
			if err == nil {
				t.Fatal("Save through symlinked parent returned no error")
			}
			if overwrite {
				got, readErr := os.ReadFile(target)
				if readErr != nil {
					t.Fatalf("read existing outside file: %v", readErr)
				}
				if string(got) != "original" {
					t.Fatalf("outside file was overwritten: %q", got)
				}
			} else if _, statErr := os.Stat(target); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("file was created outside the local storage root: %v", statErr)
			}

			entries, readDirErr := os.ReadDir(outside)
			if readDirErr != nil {
				t.Fatalf("read outside directory: %v", readDirErr)
			}
			wantEntries := 0
			if overwrite {
				wantEntries = 1
			}
			if len(entries) != wantEntries {
				t.Fatalf("outside directory has %d entries, want %d", len(entries), wantEntries)
			}
		})
	}
}

func TestLocalAllowsSymlinkedParentInsideRoot(t *testing.T) {
	l := newTestLocal(t)
	inside := filepath.Join(l.config.BasePath, "inside")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatalf("create inside directory: %v", err)
	}
	if err := os.Symlink("inside", filepath.Join(l.config.BasePath, "linked")); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	if err := l.Save(context.Background(), strings.NewReader("x"), "linked/file.txt"); err != nil {
		t.Fatalf("Save through symlinked parent inside root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inside, "file.txt")); err != nil {
		t.Fatalf("stat saved file: %v", err)
	}
}
