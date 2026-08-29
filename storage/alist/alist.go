package alist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	config "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"golang.org/x/sync/singleflight"
)

type Alist struct {
	client      *http.Client
	tokenMu     sync.RWMutex
	token       string
	lastLoginAt time.Time
	tokenFlight singleflight.Group
	baseURL     string
	loginInfo   *loginRequest
	config      config.AlistStorageConfig
	logger      *log.Logger
}

// authHeader returns the current token for use in API requests.
func (a *Alist) authHeader() string {
	a.tokenMu.RLock()
	defer a.tokenMu.RUnlock()
	return a.token
}

func (a *Alist) Init(ctx context.Context, cfg config.StorageConfig) error {
	alistConfig, ok := cfg.(*config.AlistStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast alist config")
	}
	if err := alistConfig.Validate(); err != nil {
		return err
	}
	a.config = *alistConfig
	a.baseURL = alistConfig.URL
	a.client = getHttpClient()
	a.logger = log.FromContext(ctx).WithPrefix(fmt.Sprintf("alist[%s]", alistConfig.Name))

	if alistConfig.Token != "" {
		a.tokenMu.Lock()
		a.token = alistConfig.Token
		a.tokenMu.Unlock()
		tokenCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		req, err := http.NewRequestWithContext(tokenCtx, http.MethodGet, a.baseURL+"/api/me", nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", a.authHeader())

		resp, err := a.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get alist user info: %s", resp.Status)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		var meResp meResponse
		if err := json.Unmarshal(body, &meResp); err != nil {
			return fmt.Errorf("failed to unmarshal me response: %w", err)
		}
		if meResp.Code != http.StatusOK {
			return fmt.Errorf("failed to get alist user info: %s", meResp.Message)
		}
		a.logger.Debugf("Logged in Alist as %s", meResp.Data.Username)
		return nil
	}
	a.loginInfo = &loginRequest{
		Username: alistConfig.Username,
		Password: alistConfig.Password,
	}

	if err := a.getToken(ctx); err != nil {
		return fmt.Errorf("failed to login to Alist: %w", err)
	}
	// The init login must not satisfy the refresh dedup window.
	a.tokenMu.Lock()
	a.lastLoginAt = time.Time{}
	a.tokenMu.Unlock()
	a.logger.Debug("Logged in to Alist")

	go a.refreshToken(ctx, *alistConfig)
	return nil
}

func (a *Alist) Type() storenum.StorageType {
	return storenum.Alist
}

func (a *Alist) Name() string {
	return a.config.Name
}

func (a *Alist) Save(ctx context.Context, reader io.Reader, storagePath string) error {
	a.logger.Infof("Saving file to %s", storagePath)
	candidate := a.JoinStoragePath(storagePath)
	if overwrite, _ := ctx.Value(ctxkey.OverwriteExisting).(bool); !overwrite {
		candidate = fsutil.UniquePath(a.config.BasePath, storagePath, func(c string) bool {
			return a.existsPath(ctx, c)
		}, 1000)
	}

	if err := a.mkdirAll(ctx, path.Dir(candidate)); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	resp, err := a.putFile(ctx, reader, candidate)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		status := resp.Status
		resp.Body.Close()
		// Token-only storage cannot refresh: surface the auth error.
		if a.loginInfo == nil {
			return fmt.Errorf("failed to save file to Alist: %s", status)
		}
		if err := a.getToken(ctx); err != nil {
			return fmt.Errorf("failed to refresh alist token: %w", err)
		}
		rs, seekable := reader.(io.ReadSeeker)
		if !seekable {
			a.logger.Warnf("Upload rejected with %s; reader is not seekable, cannot retry", status)
			return fmt.Errorf("failed to save file to Alist: %s (streaming reader cannot be replayed for retry)", status)
		}
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to rewind reader before retry: %w", err)
		}
		a.logger.Info("Retrying upload with refreshed token")
		resp, err = a.putFile(ctx, reader, candidate)
		if err != nil {
			return err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to save file to Alist: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var putResp putResponse
	if err := json.Unmarshal(body, &putResp); err != nil {
		return fmt.Errorf("failed to unmarshal put response: %w", err)
	}

	if putResp.Code != http.StatusOK {
		return fmt.Errorf("failed to save file to Alist: %d, %s", putResp.Code, putResp.Message)
	}

	return nil
}

// putFile performs a single PUT upload with the given token and returns the response.
func (a *Alist) putFile(ctx context.Context, reader io.Reader, storagePath string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, a.baseURL+"/api/fs/put", reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", a.authHeader())
	req.Header.Set("File-Path", url.PathEscape(storagePath))
	req.Header.Set("Content-Type", "application/octet-stream")
	if length := ctx.Value(ctxkey.ContentLength); length != nil {
		length, ok := length.(int64)
		if ok {
			req.ContentLength = length
		}
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	return resp, nil
}

func (a *Alist) JoinStoragePath(p string) string {
	return path.Join(a.config.BasePath, p)
}

// mkdirAll creates the directory and any missing parents. Alist's upload API
// returns FileNotFound when the parent directory does not exist, so callers
// must ensure it before uploading. Existing directories are skipped.
func (a *Alist) mkdirAll(ctx context.Context, dirPath string) error {
	if dirPath == "" || dirPath == "/" || dirPath == "." {
		return nil
	}
	segments := strings.Split(strings.Trim(dirPath, "/"), "/")
	tokenRefreshed := false
	for i := range segments {
		current := "/" + strings.Join(segments[:i+1], "/")
		if a.existsPath(ctx, current) {
			continue
		}
		status, code, message, err := a.mkdirRequest(ctx, current)
		if err != nil {
			return err
		}
		if (status == http.StatusUnauthorized || status == http.StatusForbidden) && !tokenRefreshed {
			// Stale token. Token-only storage cannot refresh; otherwise
			// re-login once and retry this segment (the probe above runs
			// again first, so an already-created directory is not an error).
			if a.loginInfo == nil {
				return fmt.Errorf("failed to create directory %s: %s", current, message)
			}
			if err := a.getToken(ctx); err != nil {
				return fmt.Errorf("failed to refresh alist token: %w", err)
			}
			tokenRefreshed = true
			if a.existsPath(ctx, current) {
				continue
			}
			status, code, message, err = a.mkdirRequest(ctx, current)
			if err != nil {
				return err
			}
		}
		if status != http.StatusOK {
			return fmt.Errorf("failed to create directory %s: %s", current, message)
		}
		if code != http.StatusOK {
			// The directory may have been created by a concurrent upload
			// between the probe and the request.
			if a.existsPath(ctx, current) {
				continue
			}
			return fmt.Errorf("failed to create directory %s: %d, %s", current, code, message)
		}
	}
	return nil
}

// mkdirRequest sends a single /api/fs/mkdir request and returns the HTTP
// status, the response body code and message.
func (a *Alist) mkdirRequest(ctx context.Context, p string) (int, int, string, error) {
	bodyBytes, err := json.Marshal(fsMkdirRequest{Path: p})
	if err != nil {
		return 0, 0, "", fmt.Errorf("failed to marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/fs/mkdir", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return 0, 0, "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", a.authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return 0, 0, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, 0, resp.Status, nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, "", fmt.Errorf("failed to read response body: %w", err)
	}
	var mkResp fsSimpleResponse
	if err := json.Unmarshal(data, &mkResp); err != nil {
		return 0, 0, "", fmt.Errorf("failed to unmarshal mkdir response: %w", err)
	}
	return resp.StatusCode, mkResp.Code, mkResp.Message, nil
}

func (a *Alist) Exists(ctx context.Context, storagePath string) bool {
	return a.existsPath(ctx, a.JoinStoragePath(storagePath))
}

func (a *Alist) existsPath(ctx context.Context, storagePath string) bool {
	// POST  /api/fs/get
	/*
		body:
		{
		  "path": "/t",
		  "password": "",
		  "page": 1,
		  "per_page": 0,
		  "refresh": false
		}
	*/
	body := map[string]any{
		"path":     storagePath,
		"password": "",
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		a.logger.Errorf("Failed to marshal request body: %v", err)
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/fs/get", bytes.NewBuffer(bodyBytes))
	if err != nil {
		a.logger.Errorf("Failed to create request: %v", err)
		return false
	}
	req.Header.Set("Authorization", a.authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		a.logger.Errorf("Failed to send request: %v", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		a.logger.Errorf("Failed to read response body: %v", err)
		return false
	}
	var fsGetResp fsGetResponse
	if err := json.Unmarshal(data, &fsGetResp); err != nil {
		a.logger.Errorf("Failed to unmarshal fs get response: %v", err)
		return false
	}
	if fsGetResp.Code != http.StatusOK {
		a.logger.Errorf("Failed to get file info from Alist: %d, %s", fsGetResp.Code, fsGetResp.Message)
		return false
	}
	return true

}

// Impl StorageCannotStream interface
func (a *Alist) CannotStream() string {
	return "Alist does not support chunked transfer encoding"
}

// ListFiles implements StorageListable interface
func (a *Alist) ListFiles(ctx context.Context, dirPath string) ([]storagetypes.FileInfo, error) {
	a.logger.Debugf("Listing files in directory: %s", dirPath)

	reqBody := fsListRequest{
		Path:     dirPath,
		Password: "",
		Page:     1,
		PerPage:  0, // 0 means all files
		Refresh:  false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/fs/list", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", a.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list files: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var listResp fsListResponse
	if err := json.Unmarshal(data, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list response: %w", err)
	}

	if listResp.Code != http.StatusOK {
		return nil, fmt.Errorf("failed to list files: %d, %s", listResp.Code, listResp.Message)
	}

	files := make([]storagetypes.FileInfo, 0, len(listResp.Data.Content))
	for _, item := range listResp.Data.Content {
		// Parse modified time; log failures but keep zero value on error.
		var modTime time.Time
		if item.Modified != "" {
			parsedTime, err := time.Parse(time.RFC3339, item.Modified)
			if err != nil {
				a.logger.With(
					"path", path.Join(dirPath, item.Name),
					"modified_raw", item.Modified,
				).Warnf("failed to parse modified time for file")
			} else {
				modTime = parsedTime
			}
		}

		files = append(files, storagetypes.FileInfo{
			Name:    item.Name,
			Path:    path.Join(dirPath, item.Name),
			Size:    item.Size,
			IsDir:   item.IsDir,
			ModTime: modTime,
		})
	}

	a.logger.Debugf("Found %d files in directory %s", len(files), dirPath)
	return files, nil
}

// OpenFile implements StorageReadable interface
func (a *Alist) OpenFile(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	a.logger.Debugf("Opening file: %s", filePath)

	// First, get file info to get the raw_url
	reqBody := map[string]any{
		"path":     filePath,
		"password": "",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/fs/get", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", a.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("failed to get file info: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var getResp fsGetResponse
	if err := json.Unmarshal(data, &getResp); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal get response: %w", err)
	}

	if getResp.Code != http.StatusOK {
		return nil, 0, fmt.Errorf("failed to get file info: %d, %s", getResp.Code, getResp.Message)
	}

	if getResp.Data.IsDir {
		return nil, 0, fmt.Errorf("path is a directory, not a file")
	}

	// Download the file from raw_url
	downloadURL := getResp.Data.RawURL
	if downloadURL == "" {
		// If no raw_url, construct download URL
		downloadURL = a.baseURL + "/d" + filePath
	}

	downloadReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create download request: %w", err)
	}

	downloadResp, err := a.client.Do(downloadReq)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to download file: %w", err)
	}

	if downloadResp.StatusCode != http.StatusOK {
		downloadResp.Body.Close()
		return nil, 0, fmt.Errorf("failed to download file: %s", downloadResp.Status)
	}

	a.logger.Debugf("Opened file %s, size: %d bytes", filePath, getResp.Data.Size)
	return downloadResp.Body, getResp.Data.Size, nil
}
