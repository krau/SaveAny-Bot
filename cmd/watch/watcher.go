package watch

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/fsnotify/fsnotify"
)

// Watcher watches a local directory and submits stable files to the Uploader.
//
// Write-completion detection: fsnotify emits Write events throughout a write.
// Watcher debounces per file and only uploads once the file size stays
// unchanged across a debounce window, avoiding uploads of partial files.
type Watcher struct {
	root      string
	recursive bool
	debounce  time.Duration
	uploader  *Uploader
	logger    *log.Logger

	fsw *fsnotify.Watcher

	mu      sync.Mutex
	pending map[string]*time.Timer
	// lastSize is the last observed file size, used to detect a stable write.
	lastSize map[string]int64
}

type WatcherOptions struct {
	Root      string
	Recursive bool
	Debounce  time.Duration
	Uploader  *Uploader
}

// NewWatcher creates a Watcher.
func NewWatcher(ctx context.Context, opts WatcherOptions) (*Watcher, error) {
	if opts.Debounce <= 0 {
		opts.Debounce = 2 * time.Second
	}
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root path: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("failed to stat root path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("watch path must be a directory: %s", root)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		root:      root,
		recursive: opts.Recursive,
		debounce:  opts.Debounce,
		uploader:  opts.Uploader,
		logger:    log.FromContext(ctx).WithPrefix("watcher"),
		fsw:       fsw,
		pending:   make(map[string]*time.Timer),
		lastSize:  make(map[string]int64),
	}
	return w, nil
}

// Run starts watching and blocks until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	if err := w.addDir(w.root); err != nil {
		w.fsw.Close()
		return fmt.Errorf("failed to watch root: %w", err)
	}
	w.logger.Infof("watching %s (recursive=%v, debounce=%s)", w.root, w.recursive, w.debounce)

	defer w.cleanup()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("stopping watcher")
			return nil
		case event, ok := <-w.fsw.Events:
			if !ok {
				return nil
			}
			w.handleEvent(ctx, event)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return nil
			}
			w.logger.Errorf("watch error: %v", err)
		}
	}
}

func (w *Watcher) handleEvent(ctx context.Context, event fsnotify.Event) {
	// Remove/Rename: cancel any pending upload for this path.
	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		w.cancelPending(event.Name)
		return
	}

	if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) {
		return
	}

	info, err := os.Stat(event.Name)
	if err != nil {
		// File may have been removed or moved; ignore.
		return
	}

	if info.IsDir() {
		// New directory: watch it recursively and scan files already inside.
		if event.Has(fsnotify.Create) && w.recursive {
			if err := w.addDir(event.Name); err != nil {
				w.logger.Errorf("failed to watch new dir %s: %v", event.Name, err)
			}
			w.scanExisting(ctx, event.Name)
		}
		return
	}

	w.scheduleUpload(ctx, event.Name)
}

// scheduleUpload schedules a debounced upload for a file.
func (w *Watcher) scheduleUpload(ctx context.Context, file string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if t, ok := w.pending[file]; ok {
		t.Stop()
	}
	w.pending[file] = time.AfterFunc(w.debounce, func() {
		w.maybeUpload(ctx, file)
	})
}

// maybeUpload submits the upload once the debounce window passes and the file
// size is stable; otherwise it waits another window.
func (w *Watcher) maybeUpload(ctx context.Context, file string) {
	if ctx.Err() != nil {
		return
	}

	info, err := os.Stat(file)
	if err != nil {
		w.cancelPending(file)
		return
	}
	if info.IsDir() {
		w.cancelPending(file)
		return
	}

	w.mu.Lock()
	prevSize, seen := w.lastSize[file]
	curSize := info.Size()
	if !seen || prevSize != curSize {
		// Size still changing: likely still being written, wait another window.
		w.lastSize[file] = curSize
		w.pending[file] = time.AfterFunc(w.debounce, func() {
			w.maybeUpload(ctx, file)
		})
		w.mu.Unlock()
		return
	}
	// Size stable: treat write as complete.
	delete(w.pending, file)
	delete(w.lastSize, file)
	w.mu.Unlock()

	relPath, err := filepath.Rel(w.root, file)
	if err != nil {
		w.logger.Errorf("failed to compute relative path for %s: %v", file, err)
		return
	}

	w.uploader.Submit(ctx, uploadJob{localPath: file, relPath: relPath})
}

func (w *Watcher) cancelPending(file string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if t, ok := w.pending[file]; ok {
		t.Stop()
		delete(w.pending, file)
	}
	delete(w.lastSize, file)
}

// addDir adds a directory to the watch list, recursively when enabled.
func (w *Watcher) addDir(dir string) error {
	if !w.recursive {
		return w.fsw.Add(dir)
	}
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			w.logger.Warnf("skip path %s: %v", p, err)
			return nil
		}
		if d.IsDir() {
			if addErr := w.fsw.Add(p); addErr != nil {
				w.logger.Warnf("failed to watch dir %s: %v", p, addErr)
			} else {
				w.logger.Debugf("watching dir %s", p)
			}
		}
		return nil
	})
}

// scanExisting submits files already present under dir (initial sync and new-dir backfill).
func (w *Watcher) scanExisting(ctx context.Context, dir string) {
	walkFn := func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			w.logger.Warnf("skip path %s: %v", p, err)
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if !w.recursive && p != dir {
				return fs.SkipDir
			}
			return nil
		}
		relPath, relErr := filepath.Rel(w.root, p)
		if relErr != nil {
			w.logger.Errorf("failed to compute relative path for %s: %v", p, relErr)
			return nil
		}
		w.uploader.Submit(ctx, uploadJob{localPath: p, relPath: relPath})
		return nil
	}
	if err := filepath.WalkDir(dir, walkFn); err != nil && ctx.Err() == nil {
		w.logger.Errorf("failed to scan dir %s: %v", dir, err)
	}
}

// ScanExisting triggers a one-time scan and upload of existing files under the watch root.
func (w *Watcher) ScanExisting(ctx context.Context) {
	w.logger.Info("scanning existing files for initial sync")
	w.scanExisting(ctx, w.root)
}

func (w *Watcher) cleanup() {
	w.mu.Lock()
	for _, t := range w.pending {
		t.Stop()
	}
	w.pending = make(map[string]*time.Timer)
	w.lastSize = make(map[string]int64)
	w.mu.Unlock()
	w.fsw.Close()
}
