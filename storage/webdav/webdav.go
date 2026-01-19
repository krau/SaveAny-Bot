package webdav

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	config "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/rs/xid"
)

type Webdav struct {
	config config.WebdavStorageConfig
	client *Client
	logger *log.Logger
}

func (w *Webdav) Init(ctx context.Context, cfg config.StorageConfig) error {
	webdavConfig, ok := cfg.(*config.WebdavStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast webdav config")
	}
	if err := webdavConfig.Validate(); err != nil {
		return err
	}
	w.config = *webdavConfig
	w.logger = log.FromContext(ctx).WithPrefix(fmt.Sprintf("webdav[%s]", w.config.Name))
	w.client = NewClient(w.config.URL, w.config.Username, w.config.Password, &http.Client{
		Timeout: time.Hour * 12,
	})
	return nil
}

func (w *Webdav) Type() storenum.StorageType {
	return storenum.Webdav
}

func (w *Webdav) Name() string {
	return w.config.Name
}

func (w *Webdav) JoinStoragePath(p string) string {
	return path.Join(w.config.BasePath, p)
}

func (w *Webdav) Save(ctx context.Context, r io.Reader, storagePath string) error {
	w.logger.Infof("Saving file to %s", storagePath)
	storagePath = w.JoinStoragePath(storagePath)
	ext := path.Ext(storagePath)
	base := strings.TrimSuffix(storagePath, ext)
	candidate := storagePath
	for i := 1; w.Exists(ctx, candidate); i++ {
		candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
		if i > 1000 {
			w.logger.Errorf("Too many attempts to find a unique filename for %s", storagePath)
			candidate = fmt.Sprintf("%s_%s%s", base, xid.New().String(), ext)
			break
		}
	}

	if err := w.client.MkDir(ctx, path.Dir(candidate)); err != nil {
		w.logger.Errorf("Failed to create directory %s: %v", path.Dir(candidate), err)
		return ErrFailedToCreateDirectory
	}
	if err := w.client.WriteFile(ctx, candidate, r); err != nil {
		w.logger.Errorf("Failed to write file %s: %v", candidate, err)
		return ErrFailedToWriteFile
	}
	return nil
}

func (w *Webdav) Exists(ctx context.Context, storagePath string) bool {
	w.logger.Debugf("Checking if file exists at %s", storagePath)
	exists, err := w.client.Exists(ctx, storagePath)
	if err != nil {
		w.logger.Errorf("Failed to check if file exists at %s: %v", storagePath, err)
		return false
	}
	return exists
}

// ListFiles implements storage.StorageListable
func (w *Webdav) ListFiles(ctx context.Context, dirPath string) ([]storagetypes.FileInfo, error) {
	w.logger.Infof("Listing files in %s", dirPath)

	// Join with base path
	fullPath := path.Join(w.config.BasePath, dirPath)

	responses, err := w.client.ListDir(ctx, fullPath)
	if err != nil {
		w.logger.Errorf("Failed to list directory %s: %v", fullPath, err)
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	files := make([]storagetypes.FileInfo, 0, len(responses))
	for _, resp := range responses {
		// Parse the href to get the file name
		decodedHref, err := url.PathUnescape(resp.Href)
		if err != nil {
			w.logger.Warnf("Failed to unescape href %q: %v; using original value", resp.Href, err)
			decodedHref = resp.Href
		}

		// Extract filename from href
		name := path.Base(strings.TrimSuffix(decodedHref, "/"))
		if name == "" || name == "." {
			continue
		}

		// Parse modification time
		var modTime time.Time
		if resp.Propstat.Prop.GetLastModified != "" {
			// Try RFC1123 format (standard for WebDAV)
			parsedTime, err := time.Parse(time.RFC1123, resp.Propstat.Prop.GetLastModified)
			if err != nil {
				w.logger.Warnf("Failed to parse last modified time %q for %s: %v", resp.Propstat.Prop.GetLastModified, decodedHref, err)
			} else {
				modTime = parsedTime
			}
		}

		isDir := resp.Propstat.Prop.ResourceType.IsCollection()

		filePath := strings.TrimPrefix(decodedHref, path.Join("/", strings.Trim(path.Dir(fullPath), "/")))
		filePath = strings.TrimPrefix(filePath, "/")

		fileInfo := storagetypes.FileInfo{
			Name:    name,
			Path:    path.Join(dirPath, name),
			Size:    resp.Propstat.Prop.GetContentLength,
			IsDir:   isDir,
			ModTime: modTime,
		}

		files = append(files, fileInfo)
	}

	w.logger.Debugf("Found %d files/directories in %s", len(files), dirPath)
	return files, nil
}

// OpenFile implements storage.StorageReadable
func (w *Webdav) OpenFile(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	w.logger.Infof("Opening file %s", filePath)

	// Join with base path
	fullPath := path.Join(w.config.BasePath, filePath)

	reader, size, err := w.client.ReadFile(ctx, fullPath)
	if err != nil {
		w.logger.Errorf("Failed to open file %s: %v", fullPath, err)
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}

	w.logger.Debugf("Opened file %s (size: %d bytes)", filePath, size)
	return reader, size, nil
}
