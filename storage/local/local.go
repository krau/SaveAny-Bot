package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	config "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/rs/xid"
)

type Local struct {
	config config.LocalStorageConfig
	logger *log.Logger
}

func (l *Local) Init(ctx context.Context, cfg config.StorageConfig) error {
	localConfig, ok := cfg.(*config.LocalStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast local config")
	}
	if err := localConfig.Validate(); err != nil {
		return err
	}
	l.config = *localConfig
	err := os.MkdirAll(localConfig.BasePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create local storage directory: %w", err)
	}
	l.logger = log.FromContext(ctx).WithPrefix(fmt.Sprintf("local[%s]", l.config.Name))
	return nil
}

func (l *Local) Type() storenum.StorageType {
	return storenum.Local
}

func (l *Local) Name() string {
	return l.config.Name
}

func (l *Local) JoinStoragePath(path string) string {
	return filepath.Join(l.config.BasePath, path)
}

func (l *Local) cleanStoragePath(storagePath string) (string, error) {
	clean := filepath.Clean(storagePath)
	if clean == "." {
		clean = ""
	}
	if clean != "" && !filepath.IsLocal(clean) {
		return "", fmt.Errorf("local: storage path escapes base directory: %s", storagePath)
	}
	return clean, nil
}

func (l *Local) openRoot(create bool) (*os.Root, error) {
	if create {
		if err := os.MkdirAll(l.config.BasePath, os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create local storage directory: %w", err)
		}
	}
	root, err := os.OpenRoot(l.config.BasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local storage root: %w", err)
	}
	return root, nil
}

func (l *Local) Save(ctx context.Context, r io.Reader, storagePath string) error {
	_, err := l.SaveWithPath(ctx, r, storagePath)
	return err
}

// SaveWithPath saves a file and returns its actual storage-relative path after
// conflict resolution.
func (l *Local) SaveWithPath(ctx context.Context, r io.Reader, storagePath string) (string, error) {
	l.logger.Infof("Saving file to %s", storagePath)
	clean, err := l.cleanStoragePath(storagePath)
	if err != nil {
		return "", err
	}
	if clean == "" {
		return "", fmt.Errorf("local: storage path is empty")
	}
	root, err := l.openRoot(true)
	if err != nil {
		return "", err
	}
	defer root.Close()
	overwrite, _ := ctx.Value(ctxkey.OverwriteExisting).(bool)
	if !overwrite {
		clean, err = fsutil.UniquePath("", clean, func(candidate string) (bool, error) {
			return l.existsPathE(root, candidate)
		}, 1000)
		if err != nil {
			return "", err
		}
	}
	if err := root.MkdirAll(filepath.Dir(clean), os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	// Write beside the target and rename, so a failed copy never leaves a
	// truncated file under the final name.
	tempFile, tempPath, err := l.createTempFile(root, filepath.Dir(clean), filepath.Base(clean))
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	removeTemp := true
	defer func() {
		if removeTemp {
			if err := root.Remove(tempPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				l.logger.Warnf("Failed to remove temporary file %s: %v", tempPath, err)
			}
		}
	}()
	if _, err := io.Copy(tempFile, r); err != nil {
		return "", errors.Join(err, tempFile.Close())
	}
	if err := tempFile.Chmod(0o644); err != nil {
		return "", errors.Join(err, tempFile.Close())
	}
	if err := tempFile.Close(); err != nil {
		return "", err
	}
	if !overwrite {
		// Link refuses to replace an existing file, closing the check-then-write
		// race in UniquePath.
		if err := root.Link(tempPath, clean); err != nil {
			if errors.Is(err, os.ErrExist) {
				return "", storagetypes.ErrAlreadyExists
			}
			return "", err
		}
		if err := root.Remove(tempPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		removeTemp = false
		return clean, nil
	}
	if err := root.Rename(tempPath, clean); err != nil {
		return "", err
	}
	removeTemp = false
	return clean, nil
}

func (l *Local) createTempFile(root *os.Root, dir, name string) (*os.File, string, error) {
	for range 3 {
		tempPath := filepath.Join(dir, "."+name+".part-"+xid.New().String())
		file, err := root.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return file, tempPath, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, "", err
		}
	}
	return nil, "", fmt.Errorf("failed to allocate a unique temporary file")
}

func (l *Local) Exists(ctx context.Context, storagePath string) (bool, error) {
	clean, err := l.cleanStoragePath(storagePath)
	if err != nil {
		return false, err
	}
	root, err := l.openRoot(false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer root.Close()
	return l.existsPathE(root, clean)
}

func (l *Local) existsPathE(root *os.Root, storagePath string) (bool, error) {
	clean, err := l.cleanStoragePath(storagePath)
	if err != nil {
		return false, err
	}
	if clean == "" {
		clean = "."
	}
	_, err = root.Stat(clean)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// ListFiles implements StorageListable interface
func (l *Local) ListFiles(ctx context.Context, dirPath string) ([]storagetypes.FileInfo, error) {
	clean, err := l.cleanStoragePath(dirPath)
	if err != nil {
		return nil, err
	}
	if clean == "" {
		clean = "."
	}
	root, err := l.openRoot(false)
	if err != nil {
		return nil, fmt.Errorf("failed to open local storage root: %w", err)
	}
	defer root.Close()
	dir, err := root.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}
	entries, readErr := dir.ReadDir(-1)
	if closeErr := dir.Close(); readErr == nil {
		readErr = closeErr
	}
	if readErr != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, readErr)
	}

	files := make([]storagetypes.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			l.logger.Warnf("Failed to get file info for %s: %v", entry.Name(), err)
			continue
		}

		filePath := filepath.Join(clean, entry.Name())
		if clean == "." {
			filePath = entry.Name()
		}
		files = append(files, storagetypes.FileInfo{
			Name:    entry.Name(),
			Path:    filePath,
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
		})
	}

	return files, nil
}

// OpenFile implements StorageReadable interface
func (l *Local) OpenFile(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	clean, err := l.cleanStoragePath(filePath)
	if err != nil {
		return nil, 0, err
	}

	root, err := l.openRoot(false)
	if err != nil {
		return nil, 0, err
	}
	defer root.Close()
	file, err := root.Open(clean)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	return file, stat.Size(), nil
}
