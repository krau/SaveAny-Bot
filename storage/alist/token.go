package alist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/krau/SaveAny-Bot/common"
	config "github.com/krau/SaveAny-Bot/config/storage"
)

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

func (a *Alist) refreshToken(cfg config.AlistStorageConfig) {
	tokenExp := cfg.TokenExp
	if tokenExp <= 0 {
		common.Log.Warn("Invalid token expiration time, using default value")
		tokenExp = 3600
	}
	for {
		time.Sleep(time.Duration(tokenExp) * time.Second)
		if err := a.getToken(); err != nil {
			common.Log.Errorf("Failed to refresh jwt token: %v", err)
			continue
		}
		common.Log.Info("Refreshed Alist jwt token")
	}
}
