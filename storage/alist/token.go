package alist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	config "github.com/krau/SaveAny-Bot/config/storage"
)

// minTokenRefreshInterval deduplicates login storms; it is capped by the
// configured token expiry.
const minTokenRefreshInterval = 30 * time.Second

func (a *Alist) tokenRefreshWindow() time.Duration {
	window := minTokenRefreshInterval
	if exp := time.Duration(a.config.TokenExp) * time.Second; exp > 0 && exp < window {
		window = exp
	}
	return window
}

// getToken refreshes the JWT, merging concurrent calls.
func (a *Alist) getToken(ctx context.Context) error {
	a.tokenMu.RLock()
	fresh := !a.lastLoginAt.IsZero() && time.Since(a.lastLoginAt) < a.tokenRefreshWindow()
	a.tokenMu.RUnlock()
	if fresh {
		return nil
	}
	_, err, _ := a.tokenFlight.Do("token", func() (any, error) {
		a.tokenMu.RLock()
		fresh := !a.lastLoginAt.IsZero() && time.Since(a.lastLoginAt) < a.tokenRefreshWindow()
		a.tokenMu.RUnlock()
		if fresh {
			return nil, nil
		}
		return nil, a.fetchToken(ctx)
	})
	return err
}

func (a *Alist) fetchToken(ctx context.Context) error {
	if a.loginInfo == nil {
		return fmt.Errorf("token-only alist storage cannot refresh credentials")
	}
	loginBody, err := json.Marshal(a.loginInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/auth/login", bytes.NewBuffer(loginBody))
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

	a.tokenMu.Lock()
	a.token = loginResp.Data.Token
	a.lastLoginAt = time.Now()
	a.tokenMu.Unlock()
	return nil
}

func (a *Alist) refreshToken(ctx context.Context, cfg config.AlistStorageConfig) {
	tokenExp := cfg.TokenExp
	if tokenExp <= 0 {
		a.logger.Warn("Invalid token expiration time, using default value")
		tokenExp = 3600
	}
	ticker := time.NewTicker(time.Duration(tokenExp) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.getToken(ctx); err != nil {
				a.logger.Errorf("Failed to refresh jwt token: %v", err)
				continue
			}
			a.logger.Info("Refreshed Alist jwt token")
		}
	}
}
