package local

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	configstorage "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
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

func TestLocalSaveCancellationDoesNotPublish(t *testing.T) {
	for _, overwrite := range []bool{false, true} {
		t.Run(map[bool]string{false: "new file", true: "overwrite"}[overwrite], func(t *testing.T) {
			l := newTestLocal(t)
			parent := context.Background()
			const filename = "file.txt"
			if overwrite {
				if err := l.Save(parent, strings.NewReader("original"), filename); err != nil {
					t.Fatal(err)
				}
				parent = context.WithValue(parent, ctxkey.OverwriteExisting, true)
			}
			ctx, cancel := context.WithCancel(parent)
			defer cancel()
			reads := 0
			reader := readerFunc(func(p []byte) (int, error) {
				reads++
				cancel()
				if overwrite {
					// EOF in the same read bypasses the next context check in the copy.
					return copy(p, "replacement"), io.EOF
				}
				if reads > 1 {
					return 0, io.EOF
				}
				return copy(p, "partial"), nil
			})
			if _, err := l.SaveWithPath(ctx, reader, filename); !errors.Is(err, context.Canceled) {
				t.Fatalf("SaveWithPath error = %v, want context.Canceled", err)
			}
			entries, err := os.ReadDir(l.config.BasePath)
			if err != nil {
				t.Fatal(err)
			}
			if reads != 1 || len(entries) != map[bool]int{false: 0, true: 1}[overwrite] {
				t.Fatalf("reads = %d, destination entries = %v", reads, entries)
			}
			if overwrite {
				got, err := os.ReadFile(filepath.Join(l.config.BasePath, filename))
				if err != nil || string(got) != "original" {
					t.Fatalf("original = %q, %v", got, err)
				}
			}
		})
	}
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

func TestLocalSaveRecreatesRemovedRoot(t *testing.T) {
	l := newTestLocal(t)
	if err := os.RemoveAll(l.config.BasePath); err != nil {
		t.Fatalf("remove local storage root: %v", err)
	}

	if err := l.Save(context.Background(), strings.NewReader("x"), "file.txt"); err != nil {
		t.Fatalf("Save after root removal: %v", err)
	}
	if _, err := os.Stat(filepath.Join(l.config.BasePath, "file.txt")); err != nil {
		t.Fatalf("stat saved file: %v", err)
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

func TestLocalPublishFallsBackWhenHardLinksAreUnsupported(t *testing.T) {
	l := newTestLocal(t)
	root, err := l.openRoot(true)
	if err != nil {
		t.Fatalf("open root: %v", err)
	}
	defer root.Close()

	const tempPath = ".file.txt.part-test"
	const storagePath = "file.txt"
	if err := os.WriteFile(filepath.Join(l.config.BasePath, tempPath), []byte("payload"), 0o600); err != nil {
		t.Fatalf("write temporary file: %v", err)
	}

	linkUnsupported := func(string, string) error {
		return errors.New("operation not permitted")
	}
	if err := l.publishNoReplace(root, tempPath, storagePath, linkUnsupported); err != nil {
		t.Fatalf("publishNoReplace: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(l.config.BasePath, storagePath))
	if err != nil {
		t.Fatalf("read published file: %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("published file = %q, want payload", got)
	}
	if _, err := os.Stat(filepath.Join(l.config.BasePath, tempPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary file still exists or stat failed: %v", err)
	}
}

func TestLocalPublishFallbackDoesNotOverwrite(t *testing.T) {
	l := newTestLocal(t)
	root, err := l.openRoot(true)
	if err != nil {
		t.Fatalf("open root: %v", err)
	}
	defer root.Close()

	const tempPath = ".file.txt.part-test"
	const storagePath = "file.txt"
	if err := os.WriteFile(filepath.Join(l.config.BasePath, tempPath), []byte("replacement"), 0o600); err != nil {
		t.Fatalf("write temporary file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(l.config.BasePath, storagePath), []byte("original"), 0o600); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	linkUnsupported := func(string, string) error {
		return errors.New("operation not permitted")
	}
	err = l.publishNoReplace(root, tempPath, storagePath, linkUnsupported)
	if !errors.Is(err, storagetypes.ErrAlreadyExists) {
		t.Fatalf("publishNoReplace error = %v, want ErrAlreadyExists", err)
	}

	got, err := os.ReadFile(filepath.Join(l.config.BasePath, storagePath))
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}
	if string(got) != "original" {
		t.Fatalf("existing file = %q, want original", got)
	}
}
