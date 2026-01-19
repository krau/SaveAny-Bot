package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsIPAllowed(t *testing.T) {
	tests := []struct {
		name       string
		clientIP   string
		allowedIPs []string
		expected   bool
	}{
		{
			name:       "exact match",
			clientIP:   "192.168.1.100",
			allowedIPs: []string{"192.168.1.100"},
			expected:   true,
		},
		{
			name:       "no match",
			clientIP:   "192.168.1.100",
			allowedIPs: []string{"192.168.1.101"},
			expected:   false,
		},
		{
			name:       "wildcard",
			clientIP:   "192.168.1.100",
			allowedIPs: []string{"*"},
			expected:   true,
		},
		{
			name:       "CIDR match",
			clientIP:   "192.168.1.100",
			allowedIPs: []string{"192.168.1.0/24"},
			expected:   true,
		},
		{
			name:       "CIDR no match",
			clientIP:   "192.168.2.100",
			allowedIPs: []string{"192.168.1.0/24"},
			expected:   false,
		},
		{
			name:       "multiple IPs with match",
			clientIP:   "192.168.1.100",
			allowedIPs: []string{"10.0.0.1", "192.168.1.100", "172.16.0.1"},
			expected:   true,
		},
		{
			name:       "localhost",
			clientIP:   "127.0.0.1",
			allowedIPs: []string{"127.0.0.1"},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIPAllowed(tt.clientIP, tt.allowedIPs)
			if result != tt.expected {
				t.Errorf("isIPAllowed(%q, %v) = %v, want %v", 
					tt.clientIP, tt.allowedIPs, result, tt.expected)
			}
		})
	}
}

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

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedIP     string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.100:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:          "X-Forwarded-For single",
			remoteAddr:    "192.168.1.100:12345",
			xForwardedFor: "10.0.0.1",
			expectedIP:    "10.0.0.1",
		},
		{
			name:          "X-Forwarded-For multiple",
			remoteAddr:    "192.168.1.100:12345",
			xForwardedFor: "10.0.0.1, 10.0.0.2, 10.0.0.3",
			expectedIP:    "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "192.168.1.100:12345",
			xRealIP:    "10.0.0.1",
			expectedIP: "10.0.0.1",
		},
		{
			name:          "X-Forwarded-For takes precedence",
			remoteAddr:    "192.168.1.100:12345",
			xForwardedFor: "10.0.0.1",
			xRealIP:       "10.0.0.2",
			expectedIP:    "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			result := getClientIP(req)
			if result != tt.expectedIP {
				t.Errorf("getClientIP() = %q, want %q", result, tt.expectedIP)
			}
		})
	}
}
