package alist_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/krau/SaveAny-Bot/storage/alist"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
)

// Regression: concurrent uploads hitting 401 must share a single re-login
// (singleflight) and never race on the token field.
func TestConcurrent401RetrySingleLogin(t *testing.T) {
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
	// The initial configured token is rejected; refreshed tokens succeed.
	var putAuths []string
	mux.HandleFunc("/api/fs/put", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		putAuths = append(putAuths, r.Header.Get("Authorization"))
		rejected := r.Header.Get("Authorization") == "token-0"
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if rejected {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"code": 401, "message": "unauthorized"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"code": 200, "message": "ok"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := &storconfig.AlistStorageConfig{}
	cfg.Name = "probe"
	cfg.URL = srv.URL
	cfg.Token = "token-0"
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
	var saveErrs []error
	for err := range errs {
		if err != nil {
			saveErrs = append(saveErrs, err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	t.Logf("put auths: %v, logins: %d, save errors: %v", putAuths, loginCount, saveErrs)
	if len(saveErrs) > 0 {
		t.Fatalf("Save failed: %v", saveErrs[0])
	}
	if loginCount != 1 {
		t.Fatalf("expected exactly 1 login for %d concurrent 401 retries, got %d", workers, loginCount)
	}
}
