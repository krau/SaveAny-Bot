//go:build e2e

package e2e

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestHealthAndAuth(t *testing.T) {
	status, body := (&apiClient{t: t, base: serverURL, http: http.DefaultClient}).request(http.MethodGet, "/health", nil, false)
	if status != http.StatusOK || !strings.Contains(string(body), `"status":"ok"`) {
		t.Fatalf("health = %d %s", status, body)
	}

	anon := &apiClient{t: t, base: serverURL, http: http.DefaultClient}
	if status, _ := anon.request(http.MethodPost, "/api/v1/auth/login", map[string]string{"password": "wrong"}, false); status != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d", status)
	}

	client := newClient(t)
	var me struct {
		Authenticated bool   `json:"authenticated"`
		CSRF          string `json:"csrf_token"`
	}
	if status := client.do(http.MethodGet, "/api/v1/auth/me", nil, false, &me); status != http.StatusOK || !me.Authenticated {
		t.Fatalf("me = %d %+v", status, me)
	}

	if status := client.do(http.MethodPost, "/api/v1/auth/logout", nil, true, nil); status != http.StatusNoContent {
		t.Fatalf("logout status = %d", status)
	}
	if status := client.do(http.MethodGet, "/api/v1/auth/me", nil, false, nil); status != http.StatusUnauthorized {
		t.Fatalf("me after logout = %d", status)
	}
}

func TestCSRFAndSettingsRoundTrip(t *testing.T) {
	client := sharedClient(t)

	var settings map[string]map[string]any
	if status := client.do(http.MethodGet, "/api/v1/settings", nil, false, &settings); status != http.StatusOK {
		t.Fatalf("get settings = %d", status)
	}
	core := settings["core"]
	originalLang, _ := core["lang"].(string)
	nextLang := "en"
	if originalLang == nextLang {
		nextLang = "zh-Hans"
	}
	defer func() {
		client.do(http.MethodPut, "/api/v1/settings/core", map[string]any{"lang": originalLang}, true, nil)
	}()

	if status := client.do(http.MethodPut, "/api/v1/settings/core", map[string]any{"lang": nextLang}, false, nil); status != http.StatusForbidden {
		t.Fatalf("settings update without CSRF = %d, want 403", status)
	}
	if status := client.do(http.MethodPut, "/api/v1/settings/core", map[string]any{"lang": nextLang}, true, nil); status != http.StatusOK {
		t.Fatalf("settings update with CSRF = %d", status)
	}
	if status := client.do(http.MethodGet, "/api/v1/settings", nil, false, &settings); status != http.StatusOK {
		t.Fatalf("get settings after update = %d", status)
	}
	if got, _ := settings["core"]["lang"].(string); got != nextLang {
		t.Fatalf("lang = %q, want %q", got, nextLang)
	}
}

func TestSeededEnvironment(t *testing.T) {
	client := sharedClient(t)

	var storages struct {
		Storages []struct {
			ID     uint           `json:"id"`
			Name   string         `json:"name"`
			Type   string         `json:"type"`
			Enable bool           `json:"enable"`
			Config map[string]any `json:"config"`
		} `json:"storages"`
	}
	if status := client.do(http.MethodGet, "/api/v1/storages", nil, false, &storages); status != http.StatusOK {
		t.Fatalf("list storages = %d", status)
	}
	if len(storages.Storages) != 1 || storages.Storages[0].Name != "e2e-local" {
		t.Fatalf("seeded storages = %+v", storages.Storages)
	}
	storageID := storages.Storages[0].ID

	var testResult struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if status := client.do(http.MethodPost, "/api/v1/storages/"+strconv.FormatUint(uint64(storageID), 10)+"/test", nil, true, &testResult); status != http.StatusOK || !testResult.OK {
		t.Fatalf("storage test = %d %+v", status, testResult)
	}

	var users struct {
		Users []struct {
			ChatID           int64 `json:"chat_id"`
			Enabled          bool  `json:"enabled"`
			DefaultStorageID uint  `json:"default_storage_id"`
		} `json:"users"`
	}
	if status := client.do(http.MethodGet, "/api/v1/users", nil, false, &users); status != http.StatusOK {
		t.Fatalf("list users = %d", status)
	}
	if len(users.Users) != 1 || users.Users[0].ChatID != 10001 || !users.Users[0].Enabled || users.Users[0].DefaultStorageID != storageID {
		t.Fatalf("seeded users = %+v", users.Users)
	}

	var onboarding struct {
		Steps struct {
			BotConfigured bool `json:"bot_configured"`
			StorageReady  bool `json:"storage_ready"`
			UserReady     bool `json:"user_ready"`
		} `json:"steps"`
	}
	if status := client.do(http.MethodGet, "/api/v1/onboarding", nil, false, &onboarding); status != http.StatusOK {
		t.Fatalf("onboarding = %d", status)
	}
	if onboarding.Steps.BotConfigured || !onboarding.Steps.StorageReady || !onboarding.Steps.UserReady {
		t.Fatalf("onboarding steps = %+v", onboarding.Steps)
	}
}

func TestStorageLifecycle(t *testing.T) {
	client := sharedClient(t)
	basePath := t.TempDir()

	var created struct {
		ID uint `json:"id"`
	}
	status := client.do(http.MethodPost, "/api/v1/storages", map[string]any{
		"name": "e2e-temp", "type": "local", "enable": true, "config": map[string]any{"base_path": basePath},
	}, true, &created)
	if status != http.StatusCreated || created.ID == 0 {
		t.Fatalf("create storage = %d %+v", status, created)
	}

	var testResult struct {
		OK bool `json:"ok"`
	}
	if status := client.do(http.MethodPost, "/api/v1/storages/"+strconv.FormatUint(uint64(created.ID), 10)+"/test", nil, true, &testResult); status != http.StatusOK || !testResult.OK {
		t.Fatalf("test storage = %d %+v", status, testResult)
	}

	if status := client.do(http.MethodPut, "/api/v1/storages/"+strconv.FormatUint(uint64(created.ID), 10), map[string]any{
		"name": "e2e-temp", "type": "local", "enable": false, "config": map[string]any{"base_path": basePath},
	}, true, nil); status != http.StatusOK {
		t.Fatalf("update storage = %d", status)
	}

	if status := client.do(http.MethodDelete, "/api/v1/storages/"+strconv.FormatUint(uint64(created.ID), 10), nil, true, nil); status != http.StatusNoContent {
		t.Fatalf("delete storage = %d", status)
	}
}

func TestUserLifecycle(t *testing.T) {
	client := sharedClient(t)

	var created struct {
		ID     uint  `json:"id"`
		ChatID int64 `json:"chat_id"`
	}
	status := client.do(http.MethodPost, "/api/v1/users", map[string]any{
		"chat_id": 20002, "enabled": true,
	}, true, &created)
	if status != http.StatusCreated || created.ID == 0 {
		t.Fatalf("create user = %d %+v", status, created)
	}

	if status := client.do(http.MethodPut, "/api/v1/users/"+strconv.FormatUint(uint64(created.ID), 10), map[string]any{"silent": true}, true, nil); status != http.StatusOK {
		t.Fatalf("update user = %d", status)
	}

	if status := client.do(http.MethodDelete, "/api/v1/users/"+strconv.FormatUint(uint64(created.ID), 10), nil, true, nil); status != http.StatusNoContent {
		t.Fatalf("delete user = %d", status)
	}

	var users struct {
		Users []struct {
			ChatID int64 `json:"chat_id"`
		} `json:"users"`
	}
	if status := client.do(http.MethodGet, "/api/v1/users", nil, false, &users); status != http.StatusOK {
		t.Fatalf("list users = %d", status)
	}
	for _, user := range users.Users {
		if user.ChatID == 20002 {
			t.Fatal("deleted user still listed")
		}
	}
}

func TestViewEndpoints(t *testing.T) {
	client := sharedClient(t)
	paths := []string{
		"/api/v1/settings",
		"/api/v1/storages",
		"/api/v1/users",
		"/api/v1/telegram/status",
		"/api/v1/telegram/workers",
		"/api/v1/tasks",
		"/api/v1/task-types",
		"/api/v1/system/info",
		"/api/v1/onboarding",
		"/api/v1/filesystem?path=" + dataDir,
	}
	for _, path := range paths {
		if status := client.do(http.MethodGet, path, nil, false, nil); status != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, status)
		}
	}
}

func TestSPAServing(t *testing.T) {
	index, err := os.ReadFile(filepath.Join(repoRoot, "web", "dist", "index.html"))
	if err != nil {
		t.Skip("web UI is not built; run make web")
	}
	client := newClient(t)
	status, body := client.request(http.MethodGet, "/", nil, false)
	if status != http.StatusOK || !strings.Contains(string(body), `id="app"`) {
		t.Fatalf("GET / = %d, body length %d", status, len(body))
	}
	status, body = client.request(http.MethodGet, "/tasks", nil, false)
	if status != http.StatusOK || !strings.Contains(string(body), `id="app"`) {
		t.Fatalf("SPA fallback = %d, body length %d", status, len(body))
	}
	if strings.Contains(string(index), "/assets/") {
		start := strings.Index(string(index), "/assets/")
		end := strings.Index(string(index[start:]), `"`)
		if end > 0 {
			asset := string(index[start : start+end])
			if status, _ := client.request(http.MethodGet, asset, nil, false); status != http.StatusOK {
				t.Fatalf("GET %s = %d", asset, status)
			}
		}
	}
}
