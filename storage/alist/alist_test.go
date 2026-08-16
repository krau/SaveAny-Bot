package alist_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/krau/SaveAny-Bot/storage/alist"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
)

func newAlistServer(t *testing.T) (*httptest.Server, *sync.Mutex, *int) {
	t.Helper()
	var mu sync.Mutex
	loginCount := 0
	tokenSeq := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		loginCount++
		tokenSeq++
		token := fmt.Sprintf("token-%d", tokenSeq)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 200, "message": "ok", "data": map[string]any{"token": token}})
	})
	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 200, "message": "ok", "data": map[string]any{"username": "probe"}})
	})
	mux.HandleFunc("/api/fs/put", func(w http.ResponseWriter, r *http.Request) {
		rejected := r.Header.Get("Authorization") == "token-0"
		w.Header().Set("Content-Type", "application/json")
		if rejected {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"code": 401, "message": "unauthorized"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"code": 200, "message": "ok"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &mu, &loginCount
}

// Regression: concurrent uploads hitting 401 must share a single re-login
// (singleflight) and never race on the token field. Uses username/password
// credentials because token-only storages must not refresh at all.
func TestConcurrent401RetrySingleLogin(t *testing.T) {
	srv, mu, loginCount := newAlistServer(t)

	cfg := &storconfig.AlistStorageConfig{}
	cfg.Name = "probe"
	cfg.URL = srv.URL
	cfg.Username = "user"
	cfg.Password = "pass"
	cfg.BasePath = "/probe"

	stor := &alist.Alist{}
	if err := stor.Init(t.Context(), cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	const workers = 10
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- stor.Save(t.Context(), bytes.NewReader([]byte("data")), "dir/file.txt")
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if *loginCount != 1 {
		t.Fatalf("expected exactly 1 login for %d concurrent 401 retries, got %d", workers, *loginCount)
	}
}

// A token-only storage receives 401 and must return the auth error without
// attempting a login (it has no credentials to refresh with).
func TestTokenOnlyNoLoginOn401(t *testing.T) {
	srv, mu, loginCount := newAlistServer(t)

	cfg := &storconfig.AlistStorageConfig{}
	cfg.Name = "probe"
	cfg.URL = srv.URL
	cfg.Token = "token-0"
	cfg.BasePath = "/probe"

	stor := &alist.Alist{}
	if err := stor.Init(t.Context(), cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	err := stor.Save(t.Context(), bytes.NewReader([]byte("data")), "dir/file.txt")
	if err == nil {
		t.Fatal("expected auth error from token-only storage, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 auth error, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if *loginCount != 0 {
		t.Fatalf("expected no login attempts for token-only storage, got %d", *loginCount)
	}
}
