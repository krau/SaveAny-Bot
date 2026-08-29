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

	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/storage/alist"
)

// newAlistServer starts a fake alist whose login endpoint issues sequential
// tokens and whose PUT endpoint rejects the given token (simulating an expired
// credential) while accepting refreshed ones.
func newAlistServer(t *testing.T, rejectedToken string) (*httptest.Server, *sync.Mutex, *int, *[]putRecord, map[string]bool) {
	t.Helper()
	var mu sync.Mutex
	loginCount := 0
	tokenSeq := 0
	createdDirs := map[string]bool{}
	var puts []putRecord

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
		mu.Lock()
		rejected := r.Header.Get("Authorization") == rejectedToken
		puts = append(puts, putRecord{auth: r.Header.Get("Authorization"), rejected: rejected})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if rejected {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"code": 401, "message": "unauthorized"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"code": 200, "message": "ok"})
	})
	mux.HandleFunc("/api/fs/get", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Path string `json:"path"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		defer mu.Unlock()
		exists := body.Path == "/probe" || createdDirs[body.Path]
		w.Header().Set("Content-Type", "application/json")
		if !exists {
			json.NewEncoder(w).Encode(map[string]any{"code": 500, "message": "object not found"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"code": 200, "message": "ok", "data": map[string]any{"is_dir": true}})
	})
	mux.HandleFunc("/api/fs/mkdir", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Path string `json:"path"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("Authorization") == rejectedToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"code": 401, "message": "unauthorized"})
			return
		}
		createdDirs[body.Path] = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 200, "message": "ok"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &mu, &loginCount, &puts, createdDirs
}

type putRecord struct {
	auth     string
	rejected bool
}

// Regression: concurrent uploads hitting 401 must share a single re-login
// (singleflight) and retry with the refreshed token. Init performs login #1
// (token-1); the server rejects it, so the concurrent uploads must trigger a
// second, merged login (token-2).
func TestConcurrent401RetrySingleLogin(t *testing.T) {
	srv, mu, loginCount, putAuths, _ := newAlistServer(t, "token-1")

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
		wg.Go(func() {
			errs <- stor.Save(t.Context(), bytes.NewReader([]byte("data")), "dir/file.txt")
		})
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
	// One login for init, one merged login for the 401 storm.
	if *loginCount != 2 {
		t.Fatalf("expected 2 logins (init + merged retry), got %d", *loginCount)
	}
	accepted := 0
	for _, put := range *putAuths {
		if put.rejected {
			if put.auth != "token-1" {
				t.Fatalf("expected rejected uploads to use the expired token-1, got %q", put.auth)
			}
			continue
		}
		accepted++
		if put.auth != "token-2" {
			t.Fatalf("expected accepted uploads to use the refreshed token-2, got %q", put.auth)
		}
	}
	if accepted != workers {
		t.Fatalf("expected %d accepted uploads, got %d", workers, accepted)
	}
}

// A token-only storage receives 401 and must return the auth error without
// attempting a login (it has no credentials to refresh with).
func TestTokenOnlyNoLoginOn401(t *testing.T) {
	srv, mu, loginCount, _, _ := newAlistServer(t, "token-0")

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

// Regression for issue #239: uploading when the parent directory does not
// exist made alist return FileNotFound (code 500) and the task layer retried
// until exhaustion. Save must create missing parent directories first.
func TestSaveCreatesMissingParentDirs(t *testing.T) {
	srv, mu, loginCount, _, createdDirs := newAlistServer(t, "unused")

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

	if err := stor.Save(t.Context(), bytes.NewReader([]byte("data")), "a/b/file.txt"); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, dir := range []string{"/probe/a", "/probe/a/b"} {
		if !createdDirs[dir] {
			t.Fatalf("expected directory %s to be created", dir)
		}
	}
	if *loginCount != 1 {
		t.Fatalf("expected only the init login, got %d", *loginCount)
	}
}
