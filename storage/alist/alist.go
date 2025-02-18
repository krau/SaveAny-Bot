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
	config    config.AlistConfig
}

func (a *Alist) Init(model types.StorageModel) error {
	var alistConfig config.AlistConfig
	if err := json.Unmarshal([]byte(model.Config), &alistConfig); err != nil {
		return fmt.Errorf("failed to unmarshal alist config: %w", err)
	}
	a.config = alistConfig
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

	go a.refreshToken(alistConfig)
	return nil
}

func (a *Alist) Type() types.StorageType {
	return types.StorageTypeAlist
}

func (a *Alist) Save(ctx context.Context, filePath, storagePath string) error {
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
	req.Header.Set("As-Task", "true")
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
