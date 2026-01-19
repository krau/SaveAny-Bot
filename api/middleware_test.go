package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware_NoAuth(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()

	// Apply middleware
	authMiddleware(handler).ServeHTTP(rec, req)

	// When no token is configured, request should succeed or be unauthorized
	if rec.Code != http.StatusOK && rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 200 or 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_HealthCheck(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create request to health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	// Apply middleware
	authMiddleware(handler).ServeHTTP(rec, req)

	// Health check should always work without auth
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for health check, got %d", rec.Code)
	}
}
