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
	"time"

	"github.com/charmbracelet/log"
	config "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type Alist struct {
	client    *http.Client
	token     string
	baseURL   string
	loginInfo *loginRequest
	config    config.AlistStorageConfig
	logger    *log.Logger
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
		a.token = alistConfig.Token
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/api/me", nil)
		if err != nil {
			a.logger.Fatalf("Failed to create request: %v", err)
			return err
		}
		req.Header.Set("Authorization", a.token)

		resp, err := a.client.Do(req)
		if err != nil {
			a.logger.Fatalf("Failed to send request: %v", err)
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			a.logger.Fatalf("Failed to get alist user info: %s", resp.Status)
			return err
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			a.logger.Fatalf("Failed to read response body: %v", err)
			return err
		}
		var meResp meResponse
		if err := json.Unmarshal(body, &meResp); err != nil {
			a.logger.Fatalf("Failed to unmarshal me response: %v", err)
			return err
		}
		if meResp.Code != http.StatusOK {
			a.logger.Fatalf("Failed to get alist user info: %s", meResp.Message)
			return err
		}
		a.logger.Debugf("Logged in Alist as %s", meResp.Data.Username)
		return nil
	}
	a.loginInfo = &loginRequest{
		Username: alistConfig.Username,
		Password: alistConfig.Password,
	}

	if err := a.getToken(ctx); err != nil {
		a.logger.Fatalf("Failed to login to Alist: %v", err)
		return err
	}
	a.logger.Debug("Logged in to Alist")

	go a.refreshToken(*alistConfig)
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

	ext := path.Ext(storagePath)
	base := strings.TrimSuffix(storagePath, ext)
	candidate := storagePath
	for i := 1; a.Exists(ctx, candidate); i++ {
		candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, a.baseURL+"/api/fs/put", reader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", a.token)
	req.Header.Set("File-Path", url.PathEscape(candidate))
	req.Header.Set("Content-Type", "application/octet-stream")
	if length := ctx.Value(ctxkey.ContentLength); length != nil {
		length, ok := length.(int64)
		if ok {
			req.ContentLength = length
		}
	}

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

func (a *Alist) JoinStoragePath(p string) string {
	return path.Join(a.config.BasePath, p)
}

func (a *Alist) Exists(ctx context.Context, storagePath string) bool {
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
	req.Header.Set("Authorization", a.token)
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
