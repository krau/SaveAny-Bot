package webdav

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	config "github.com/krau/SaveAny-Bot/config/storage"
)

func TestInitConfiguresSeparateResponseHeaderTimeouts(t *testing.T) {
	tests := []struct {
		name            string
		uploadTimeout   int
		wantUploadLimit time.Duration
	}{
		{name: "disabled", uploadTimeout: 0, wantUploadLimit: 0},
		{name: "configured", uploadTimeout: 7, wantUploadLimit: 7 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := new(Webdav)
			err := storage.Init(t.Context(), &config.WebdavStorageConfig{
				URL:                         "http://example.com",
				Username:                    "user",
				Password:                    "password",
				BasePath:                    "/",
				UploadResponseHeaderTimeout: tt.uploadTimeout,
			})
			if err != nil {
				t.Fatal(err)
			}

			metadataTransport, ok := storage.client.httpClient.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("metadata transport = %T, want *http.Transport", storage.client.httpClient.Transport)
			}
			if metadataTransport.ResponseHeaderTimeout != metadataResponseHeaderTimeout {
				t.Fatalf("metadata timeout = %v, want %v", metadataTransport.ResponseHeaderTimeout, metadataResponseHeaderTimeout)
			}

			uploadTransport, ok := storage.client.uploadHTTPClient.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("upload transport = %T, want *http.Transport", storage.client.uploadHTTPClient.Transport)
			}
			if uploadTransport.ResponseHeaderTimeout != tt.wantUploadLimit {
				t.Fatalf("upload timeout = %v, want %v", uploadTransport.ResponseHeaderTimeout, tt.wantUploadLimit)
			}
		})
	}
}

func TestWriteFileUsesUploadHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == string(WebdavMethodPut) {
			_, _ = io.Copy(io.Discard, r.Body)
		}
		time.Sleep(100 * time.Millisecond)
		if r.Method == string(WebdavMethodPut) {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusMultiStatus)
	}))
	t.Cleanup(server.Close)

	metadataClient := &http.Client{Transport: &http.Transport{
		ResponseHeaderTimeout: 20 * time.Millisecond,
	}}
	uploadClient := &http.Client{Transport: &http.Transport{
		ResponseHeaderTimeout: 5 * time.Second,
	}}
	client := NewClientWithUploadHTTPClient(server.URL, "", "", metadataClient, uploadClient)

	if err := client.WriteFile(t.Context(), "large.bin", strings.NewReader("content")); err != nil {
		t.Fatalf("WriteFile() error = %v; upload timeout should allow a delayed response", err)
	}

	if _, err := client.Exists(t.Context(), "large.bin"); err == nil {
		t.Fatal("Exists() succeeded; metadata requests should still use the short timeout")
	} else if !strings.Contains(err.Error(), "timeout awaiting response headers") {
		t.Fatalf("Exists() error = %v, want a response header timeout", err)
	}
}
