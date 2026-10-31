package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/krau/SaveAny-Bot/database"
)

const (
	sessionCookieName = "sab_session"
	sessionTTL        = 7 * 24 * time.Hour
	csrfHeader        = "X-CSRF-Token"
)

type sessionCtxKey struct{}

func sessionFromContext(ctx context.Context) *database.WebSession {
	session, _ := ctx.Value(sessionCtxKey{}).(*database.WebSession)
	return session
}

type loginLimiter struct {
	mu       sync.Mutex
	limiter  map[string]*rate.Limiter
	lastSeen map[string]time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{
		limiter:  make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
	}
}

func (l *loginLimiter) cleanup(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, seen := range l.lastSeen {
		if seen.Before(cutoff) {
			delete(l.lastSeen, key)
			delete(l.limiter, key)
		}
	}
}

func (l *loginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lastSeen[key] = time.Now()
	lim, ok := l.limiter[key]
	if !ok {
		lim = rate.NewLimiter(rate.Every(12*time.Second), 5)
		l.limiter[key] = lim
	}
	return lim.Allow()
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *server) isPublic(r *http.Request) bool {
	if r.URL.Path == "/health" {
		return true
	}
	return r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login"
}

func (s *server) bearerToken() string {
	return s.deps.Settings.Config().Web.APIToken
}

func (s *server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.isPublic(r) {
			next.ServeHTTP(w, r)
			return
		}

		if token := s.bearerToken(); token != "" {
			header := r.Header.Get("Authorization")
			if len(header) > 7 && strings.EqualFold(header[:7], "Bearer ") {
				if subtle.ConstantTimeCompare([]byte(header[7:]), []byte(token)) == 1 {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		session, err := database.GetWebSession(r.Context(), cookie.Value)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "session expired")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get(csrfHeader)), []byte(session.CSRF)) != 1 {
				WriteError(w, http.StatusForbidden, "csrf_failed", "invalid csrf token")
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionCtxKey{}, session)))
	})
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !s.logins.allow(host) {
		WriteError(w, http.StatusTooManyRequests, "too_many_attempts", "too many login attempts")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if !s.deps.Settings.VerifyAdminPassword(body.Password) {
		WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid password")
		return
	}
	id, err := randomToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create session")
		return
	}
	csrf, err := randomToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create session")
		return
	}
	session := &database.WebSession{
		ID:        id,
		CSRF:      csrf,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	if err := database.CreateWebSession(r.Context(), session); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", "failed to store session")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.deps.Settings.Config().Web.SecureCookie,
	})
	WriteJSON(w, http.StatusOK, map[string]string{"csrf_token": csrf})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if session := sessionFromContext(r.Context()); session != nil {
		_ = database.DeleteWebSession(r.Context(), session.ID)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) me(w http.ResponseWriter, r *http.Request) {
	session := sessionFromContext(r.Context())
	if session == nil {
		WriteJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"csrf_token":    session.CSRF,
	})
}

func (s *server) changePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if err := s.deps.Settings.ChangeAdminPassword(body.CurrentPassword, body.NewPassword); err != nil {
		WriteError(w, http.StatusBadRequest, "password_change_failed", err.Error())
		return
	}
	keepID := ""
	if session := sessionFromContext(r.Context()); session != nil {
		keepID = session.ID
	}
	if err := database.DeleteOtherWebSessions(r.Context(), keepID); err != nil {
		WriteError(w, http.StatusInternalServerError, "session_cleanup_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *server) sessionCleanup(ctx context.Context) {
	_ = database.DeleteExpiredWebSessions(ctx)
	s.logins.cleanup(24 * time.Hour)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = database.DeleteExpiredWebSessions(ctx)
			s.logins.cleanup(24 * time.Hour)
		}
	}
}
