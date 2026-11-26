package alist

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetHTTPClientResponseHeaderTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{name: "disabled", timeout: 0},
		{name: "configured", timeout: 7 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := getHTTPClient(tt.timeout)
			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("transport = %T, want *http.Transport", client.Transport)
			}
			if transport.ResponseHeaderTimeout != tt.timeout {
				t.Fatalf("ResponseHeaderTimeout = %v, want %v", transport.ResponseHeaderTimeout, tt.timeout)
			}
		})
	}
}

func TestPutFileUsesUploadHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	storage := &Alist{
		baseURL: server.URL,
		client: &http.Client{Transport: &http.Transport{
			ResponseHeaderTimeout: 20 * time.Millisecond,
		}},
		uploadClient: &http.Client{Transport: &http.Transport{
			ResponseHeaderTimeout: 5 * time.Second,
		}},
	}

	resp, err := storage.putFile(t.Context(), strings.NewReader("content"), "/large.bin")
	if err != nil {
		t.Fatalf("putFile() error = %v; upload timeout should allow a delayed response", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("putFile() status = %s, want 200 OK", resp.Status)
	}
}
