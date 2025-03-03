package alist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/types"
)

type Alist struct {
	client    *http.Client
	token     string
	baseURL   string
	loginInfo *loginRequest
	config    config.AlistStorageConfig
}

func (a *Alist) Init(cfg config.StorageConfig) error {
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
	if alistConfig.Token != "" {
		a.token = alistConfig.Token
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/api/me", nil)
		if err != nil {
			logger.L.Fatalf("Failed to create request: %v", err)
			return err
		}
		req.Header.Set("Authorization", a.token)

		resp, err := a.client.Do(req)
		if err != nil {
			logger.L.Fatalf("Failed to send request: %v", err)
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			logger.L.Fatalf("Failed to get alist user info: %s", resp.Status)
			return err
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.L.Fatalf("Failed to read response body: %v", err)
			return err
		}
		var meResp meResponse
		if err := json.Unmarshal(body, &meResp); err != nil {
			logger.L.Fatalf("Failed to unmarshal me response: %v", err)
			return err
		}
		if meResp.Code != http.StatusOK {
			logger.L.Fatalf("Failed to get alist user info: %s", meResp.Message)
			return err
		}
		logger.L.Debugf("Logged in Alist as %s", meResp.Data.Username)
		return nil
	}
	a.loginInfo = &loginRequest{
		Username: alistConfig.Username,
		Password: alistConfig.Password,
	}

	if err := a.getToken(); err != nil {
		logger.L.Fatalf("Failed to login to Alist: %v", err)
		return err
	}
	logger.L.Debug("Logged in to Alist")

	go a.refreshToken(*alistConfig)
	return nil
}

func (a *Alist) Type() types.StorageType {
	return types.StorageTypeAlist
}

func (a *Alist) Name() string {
	return a.config.Name
}

func (a *Alist) Save(ctx context.Context, filePath, storagePath string) error {
	logger.L.Infof("Saving file %s to %s", filePath, storagePath)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	filestat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, a.baseURL+"/api/fs/put", file)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", a.token)
	req.Header.Set("File-Path", url.PathEscape(storagePath))
	// req.Header.Set("As-Task", "true")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = filestat.Size()

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
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

func (a *Alist) JoinStoragePath(task types.Task) string {
	return path.Join(a.config.BasePath, task.StoragePath)
}

type uploadStream struct {
	ctx         context.Context
	client      *http.Client
	token       string
	storagePath string
	baseURL     string
	pr          *io.PipeReader
	pw          *io.PipeWriter
	errChan     chan error
	once        sync.Once
}

func (us *uploadStream) Write(p []byte) (int, error) {
	return us.pw.Write(p)
}

func (us *uploadStream) Close() error {
	var uploadErr error
	us.once.Do(func() {
		if err := us.pw.Close(); err != nil {
			uploadErr = fmt.Errorf("failed to close pipe writer: %w", err)
			return
		}

		if err := <-us.errChan; err != nil {
			uploadErr = err
		}
	})
	return uploadErr
}

func (a *Alist) NewUploadStream(ctx context.Context, storagePath string) (io.WriteCloser, error) {
	if a.token == "" {
		if err := a.getToken(); err != nil {
			return nil, fmt.Errorf("not logged in to Alist: %w", err)
		}
	}

	pr, pw := io.Pipe()

	us := &uploadStream{
		ctx:         ctx,
		client:      a.client,
		token:       a.token,
		storagePath: storagePath,
		baseURL:     a.baseURL,
		pr:          pr,
		pw:          pw,
		errChan:     make(chan error, 1),
	}

	go func() {
		defer close(us.errChan)

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, a.baseURL+"/api/fs/put", pr)
		if err != nil {
			us.errChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Authorization", a.token)
		req.Header.Set("File-Path", url.PathEscape(storagePath))
		// req.Header.Set("As-Task", "true")
		req.Header.Set("Content-Type", "application/octet-stream")

		resp, err := a.client.Do(req)
		if err != nil {
			us.errChan <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			us.errChan <- fmt.Errorf("failed to upload file, status code: %d, response: %s", resp.StatusCode, string(body))
			return
		}

		us.errChan <- nil
	}()

	return us, nil
}
