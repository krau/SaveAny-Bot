//go:build e2e

package e2e

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

type taskInfo struct {
	TaskID string `json:"task_id"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Items  []struct {
		Name   string `json:"name"`
		Path   string `json:"path"`
		Status string `json:"status"`
	} `json:"items"`
}

func TestUploadTask(t *testing.T) {
	content := []byte("SaveAny-Bot web upload E2E\n")
	client := sharedClient(t)
	storageID := storageIDByName(t, client, "e2e-local")
	query := url.Values{
		"storage_id": {strconv.FormatUint(uint64(storageID), 10)},
		"path":       {"e2e-upload"},
		"filename":   {"../uploaded.txt"},
	}
	var created struct {
		TaskID string `json:"task_id"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}
	status := client.doRaw(
		http.MethodPost,
		"/api/v1/uploads?"+query.Encode(),
		content,
		"application/octet-stream",
		true,
		&created,
	)
	if status != http.StatusCreated || created.TaskID == "" || created.Type != "upload" {
		t.Fatalf("create upload = %d %+v", status, created)
	}

	info := waitForTask(t, client, created.TaskID)
	if info.Status != "completed" || info.Type != "upload" {
		t.Fatalf("upload task = %+v", info)
	}
	got, err := os.ReadFile(filepath.Join(dataDir, "downloads", "e2e-upload", "uploaded.txt"))
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("uploaded content = %q, want %q", got, content)
	}
	if status := client.do(http.MethodDelete, "/api/v1/tasks/"+created.TaskID, nil, true, nil); status != http.StatusNoContent {
		t.Fatalf("delete upload task = %d", status)
	}
}

func TestUploadSizeLimit(t *testing.T) {
	client := sharedClient(t)
	storageID := storageIDByName(t, client, "e2e-local")

	var settings map[string]map[string]any
	if status := client.do(http.MethodGet, "/api/v1/settings", nil, false, &settings); status != http.StatusOK {
		t.Fatalf("get settings = %d", status)
	}
	original, ok := settings["web"]["max_upload_size_mb"].(float64)
	if !ok || original < 1 {
		t.Fatalf("web.max_upload_size_mb = %#v", settings["web"]["max_upload_size_mb"])
	}
	defer func() {
		client.do(http.MethodPut, "/api/v1/settings/web", map[string]any{
			"max_upload_size_mb": original,
		}, true, nil)
	}()
	if status := client.do(http.MethodPut, "/api/v1/settings/web", map[string]any{
		"max_upload_size_mb": 1,
	}, true, nil); status != http.StatusOK {
		t.Fatalf("set upload limit = %d", status)
	}

	query := url.Values{
		"storage_id": {strconv.FormatUint(uint64(storageID), 10)},
		"path":       {"e2e-upload-limit"},
		"filename":   {"too-large.bin"},
	}
	status := client.doRaw(
		http.MethodPost,
		"/api/v1/uploads?"+query.Encode(),
		make([]byte, (1<<20)+1),
		"application/octet-stream",
		true,
		nil,
	)
	if status != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized upload = %d, want 413", status)
	}
}

func TestDirectLinksTask(t *testing.T) {
	content := []byte("SaveAny-Bot direct-links E2E\n")
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sample.txt" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write(content); err != nil {
			t.Errorf("write source response: %v", err)
		}
	}))
	defer source.Close()

	client := sharedClient(t)
	storageID := storageIDByName(t, client, "e2e-local")
	var created struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	status := client.do(http.MethodPost, "/api/v1/tasks", map[string]any{
		"type":       "directlinks",
		"storage_id": storageID,
		"path":       "e2e",
		"params":     map[string]any{"urls": []string{source.URL + "/sample.txt"}},
	}, true, &created)
	if status != http.StatusCreated || created.TaskID == "" {
		t.Fatalf("create task = %d %+v", status, created)
	}

	info := waitForTask(t, client, created.TaskID)
	if info.Status != "completed" {
		t.Fatalf("task status = %q, want completed; items = %+v", info.Status, info.Items)
	}

	path := filepath.Join(dataDir, "downloads", "e2e", "sample.txt")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("downloaded content = %q, want %q", got, content)
	}

	var list struct {
		Tasks []taskInfo `json:"tasks"`
		Total int        `json:"total"`
	}
	if status := client.do(http.MethodGet, "/api/v1/tasks", nil, false, &list); status != http.StatusOK {
		t.Fatalf("list tasks = %d", status)
	}
	if list.Total != 1 || len(list.Tasks) != 1 || list.Tasks[0].TaskID != created.TaskID {
		t.Fatalf("task list = %+v", list)
	}

	if status := client.do(http.MethodDelete, "/api/v1/tasks/"+created.TaskID, nil, true, nil); status != http.StatusNoContent {
		t.Fatalf("delete task = %d", status)
	}
}

func TestTaskValidation(t *testing.T) {
	client := sharedClient(t)
	storageID := storageIDByName(t, client, "e2e-local")

	if status := client.do(http.MethodPost, "/api/v1/tasks", map[string]any{
		"type": "directlinks", "storage_id": 999999, "params": map[string]any{"urls": []string{"http://example.com/a"}},
	}, true, nil); status != http.StatusBadRequest {
		t.Fatalf("unknown storage status = %d, want 400", status)
	}

	if status := client.do(http.MethodPost, "/api/v1/tasks", map[string]any{
		"type": "tgfiles", "storage_id": storageID, "params": map[string]any{"message_links": []string{"https://t.me/example/1"}},
	}, true, nil); status != http.StatusServiceUnavailable {
		t.Fatalf("tgfiles without client status = %d, want 503", status)
	}

	if status := client.do(http.MethodPost, "/api/v1/tasks", map[string]any{
		"type": "bogus", "storage_id": storageID, "params": map[string]any{},
	}, true, nil); status != http.StatusBadRequest {
		t.Fatalf("unsupported type status = %d, want 400", status)
	}
}

func waitForTask(t *testing.T, client *apiClient, id string) taskInfo {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		var info taskInfo
		if status := client.do(http.MethodGet, "/api/v1/tasks/"+id, nil, false, &info); status != http.StatusOK {
			t.Fatalf("get task %s = %d", id, status)
		}
		switch info.Status {
		case "completed", "partial", "failed", "cancelled":
			return info
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %s did not finish; last status %q", id, info.Status)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
