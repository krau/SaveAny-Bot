package alist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
)

type Alist struct {
	client    *http.Client
	token     string
	basePath  string
	baseURL   string
	loginInfo *loginRequest
}

var (
	ErrAlistLoginFailed = errors.New("failed to login to Alist")
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
	} `json:"data"`
}

type putResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Task struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			State    int    `json:"state"`
			Status   string `json:"status"`
			Progress int    `json:"progress"`
			Error    string `json:"error"`
		} `json:"task"`
	} `json:"data"`
}

func (a *Alist) getToken() error {
	loginBody, err := json.Marshal(a.loginInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, a.baseURL+"/api/auth/login", bytes.NewBuffer(loginBody))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	var loginResp loginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to unmarshal login response: %w", err)
	}

	if loginResp.Code != http.StatusOK {
		return fmt.Errorf("%w: %s", ErrAlistLoginFailed, loginResp.Message)
	}

	a.token = loginResp.Data.Token
	return nil
}

func (a *Alist) refreshToken() {
	for {
		time.Sleep(time.Duration(config.Cfg.Storage.Alist.TokenExp) * time.Second)
		if err := a.getToken(); err != nil {
			logger.L.Errorf("Failed to refresh jwt token: %v", err)
			continue
		}
		logger.L.Info("Refreshed Alist jwt token")
	}
}

func (a *Alist) Init() {
	a.basePath = config.Cfg.Storage.Alist.BasePath
	a.baseURL = config.Cfg.Storage.Alist.URL
	a.client = &http.Client{
		Timeout: 12 * time.Hour,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	a.loginInfo = &loginRequest{
		Username: config.Cfg.Storage.Alist.Username,
		Password: config.Cfg.Storage.Alist.Password,
	}

	if err := a.getToken(); err != nil {
		logger.L.Fatalf("Failed to login to Alist: %v", err)
		os.Exit(1)
	}
	logger.L.Debug("Logged in to Alist")

	go a.refreshToken()
}

func (a *Alist) Save(ctx context.Context, filePath, storagePath string) error {
	storagePath = path.Join(a.basePath, storagePath)
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
