package alist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/krau/SaveAny-Bot/common"
	config "github.com/krau/SaveAny-Bot/config/storage"
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
			common.Log.Fatalf("Failed to create request: %v", err)
			return err
		}
		req.Header.Set("Authorization", a.token)

		resp, err := a.client.Do(req)
		if err != nil {
			common.Log.Fatalf("Failed to send request: %v", err)
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			common.Log.Fatalf("Failed to get alist user info: %s", resp.Status)
			return err
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			common.Log.Fatalf("Failed to read response body: %v", err)
			return err
		}
		var meResp meResponse
		if err := json.Unmarshal(body, &meResp); err != nil {
			common.Log.Fatalf("Failed to unmarshal me response: %v", err)
			return err
		}
		if meResp.Code != http.StatusOK {
			common.Log.Fatalf("Failed to get alist user info: %s", meResp.Message)
			return err
		}
		common.Log.Debugf("Logged in Alist as %s", meResp.Data.Username)
		return nil
	}
	a.loginInfo = &loginRequest{
		Username: alistConfig.Username,
		Password: alistConfig.Password,
	}

	if err := a.getToken(); err != nil {
		common.Log.Fatalf("Failed to login to Alist: %v", err)
		return err
	}
	common.Log.Debug("Logged in to Alist")

	go a.refreshToken(*alistConfig)
	return nil
}

func (a *Alist) Type() types.StorageType {
	return types.StorageTypeAlist
}

func (a *Alist) Name() string {
	return a.config.Name
}

func (a *Alist) Save(ctx context.Context, reader io.Reader, storagePath string) error {
	common.Log.Infof("Saving file to %s", storagePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, a.baseURL+"/api/fs/put", reader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", a.token)
	req.Header.Set("File-Path", url.PathEscape(storagePath))
	req.Header.Set("Content-Type", "application/octet-stream")
	if length := ctx.Value(types.ContextKeyContentLength); length != nil {
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

func (a *Alist) NotSupportStream() string {
	return "Alist does not support chunked transfer encoding"
}

func (a *Alist) JoinStoragePath(task types.Task) string {
	return path.Join(a.config.BasePath, task.StoragePath)
}
