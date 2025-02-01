package alist

import (
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

	"github.com/imroc/req/v3"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
)

type Alist struct{}

var (
	basePath  string
	baseUrl   string
	reqClient *req.Client
	loginReq  *loginRequset

	ErrAlistLoginFailed = errors.New("failed to login to Alist")
)

type loginRequset struct {
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

func getToken() (string, error) {
	resp, err := reqClient.R().SetBodyJsonMarshal(loginReq).Post("/api/auth/login")
	if err != nil {
		return "", err
	}
	var loginResp loginResponse
	if err := json.Unmarshal(resp.Bytes(), &loginResp); err != nil {
		return "", err
	}
	if loginResp.Code != http.StatusOK {
		return "", fmt.Errorf("%w: %s", ErrAlistLoginFailed, loginResp.Message)
	}
	return loginResp.Data.Token, nil
}

func refreshToken(client *req.Client) {
	for {
		time.Sleep(time.Duration(config.Cfg.Storage.Alist.TokenExp) * time.Second)
		token, err := getToken()
		if err != nil {
			logger.L.Errorf("Failed to refresh jwt token: %v", err)
			continue
		}
		client.SetCommonHeader("Authorization", token)
		logger.L.Info("Refreshed Alist jwt token")
	}
}

func (a *Alist) Init() {
	basePath = config.Cfg.Storage.Alist.BasePath
	baseUrl = config.Cfg.Storage.Alist.URL
	reqClient = req.C().SetTLSHandshakeTimeout(time.Second * 10).SetBaseURL(baseUrl).SetTimeout(time.Hour * 24)
	loginReq = &loginRequset{
		Username: config.Cfg.Storage.Alist.Username,
		Password: config.Cfg.Storage.Alist.Password,
	}
	token, err := getToken()
	if err != nil {
		logger.L.Fatalf("Failed to login to Alist: %v", err)
		os.Exit(1)
	}
	logger.L.Debug("Logged in to Alist")
	reqClient.SetCommonHeader("Authorization", token)
	go refreshToken(reqClient)
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

func (a *Alist) Save(ctx context.Context, filePath, storagePath string) error {
	storagePath = path.Join(basePath, storagePath)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	resp, err := reqClient.R().
		SetContext(ctx).
		SetBodyBytes(fileBytes).
		SetHeaders(map[string]string{
			"File-Path": url.QueryEscape(storagePath),
			"As-Task":   "true",
		}).Put("/api/fs/put")
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to save file to Alist: %s", resp.Status)
	}
	var putResp putResponse
	if err := json.Unmarshal(resp.Bytes(), &putResp); err != nil {
		return fmt.Errorf("failed to unmarshal put response: %v", err)
	}
	if putResp.Code != http.StatusOK {
		return fmt.Errorf("failed to save file to Alist: %d, %s", putResp.Code, putResp.Message)
	}
	return nil
}
