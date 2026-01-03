package aria2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		secret  string
		wantErr bool
	}{
		{
			name:    "valid client",
			url:     "http://localhost:6800/jsonrpc",
			secret:  "test-secret",
			wantErr: false,
		},
		{
			name:    "valid client without secret",
			url:     "http://localhost:6800/jsonrpc",
			secret:  "",
			wantErr: false,
		},
		{
			name:    "invalid empty url",
			url:     "",
			secret:  "test-secret",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.url, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client")
			}
		})
	}
}

func TestClient_AddURI(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify method
		if req.Method != "aria2.addUri" {
			t.Errorf("Expected method aria2.addUri, got %s", req.Method)
		}

		// Send response
		resp := rpcResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`"2089b05ecca3d829"`),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	gid, err := client.AddURI(ctx, []string{"http://example.com/file.txt"}, nil)
	if err != nil {
		t.Fatalf("AddURI() error = %v", err)
	}

	if gid != "2089b05ecca3d829" {
		t.Errorf("Expected gid 2089b05ecca3d829, got %s", gid)
	}
}

func TestClient_TellStatus(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify method
		if req.Method != "aria2.tellStatus" {
			t.Errorf("Expected method aria2.tellStatus, got %s", req.Method)
		}

		// Send response
		status := Status{
			GID:             "2089b05ecca3d829",
			Status:          "active",
			TotalLength:     "1024000",
			CompletedLength: "512000",
			DownloadSpeed:   "102400",
			Files:           []File{},
		}
		result, _ := json.Marshal(status)

		resp := rpcResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result:  result,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	status, err := client.TellStatus(ctx, "2089b05ecca3d829")
	if err != nil {
		t.Fatalf("TellStatus() error = %v", err)
	}

	if status.GID != "2089b05ecca3d829" {
		t.Errorf("Expected gid 2089b05ecca3d829, got %s", status.GID)
	}

	if status.Status != "active" {
		t.Errorf("Expected status active, got %s", status.Status)
	}

	if !status.IsDownloadActive() {
		t.Error("Expected download to be active")
	}
}

func TestClient_WithSecret(t *testing.T) {
	expectedSecret := "my-secret-token"

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify secret token is included in params
		if len(req.Params) == 0 {
			t.Error("Expected params to contain secret token")
		} else {
			token, ok := req.Params[0].(string)
			if !ok || token != "token:"+expectedSecret {
				t.Errorf("Expected token:%s, got %v", expectedSecret, req.Params[0])
			}
		}

		// Send response
		version := Version{
			Version:         "1.36.0",
			EnabledFeatures: []string{"Async DNS", "BitTorrent", "HTTP", "HTTPS"},
		}
		result, _ := json.Marshal(version)

		resp := rpcResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result:  result,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, expectedSecret)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	version, err := client.GetVersion(ctx)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	if version.Version != "1.36.0" {
		t.Errorf("Expected version 1.36.0, got %s", version.Version)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	// Create a mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rpcResponse{
			Jsonrpc: "2.0",
			ID:      "1",
			Result:  json.RawMessage(`"OK"`),
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = client.GetVersion(ctx)
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
}

func TestClient_RPCError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp := rpcResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &rpcError{
				Code:    1,
				Message: "Unauthorized",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetVersion(ctx)
	if err == nil {
		t.Error("Expected RPC error, got nil")
	}

	var rpcErr *rpcError
	if !errors.As(err, &rpcErr) {
		t.Errorf("Expected rpcError, got %T", err)
	}
}

func TestStatus_DownloadStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		check  func(*Status) bool
	}{
		{
			name:   "active",
			status: "active",
			check:  (*Status).IsDownloadActive,
		},
		{
			name:   "waiting",
			status: "waiting",
			check:  (*Status).IsDownloadWaiting,
		},
		{
			name:   "paused",
			status: "paused",
			check:  (*Status).IsDownloadPaused,
		},
		{
			name:   "error",
			status: "error",
			check:  (*Status).IsDownloadError,
		},
		{
			name:   "complete",
			status: "complete",
			check:  (*Status).IsDownloadComplete,
		},
		{
			name:   "removed",
			status: "removed",
			check:  (*Status).IsDownloadRemoved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Status{Status: tt.status}
			if !tt.check(s) {
				t.Errorf("Expected status %s check to return true", tt.status)
			}
		})
	}
}
