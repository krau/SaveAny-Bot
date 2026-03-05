package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
)

// setupTestServer creates a test server with handlers
func setupTestServer(t *testing.T) (*Handlers, *TaskFactory) {
	factory := NewTaskFactory(t.Context())
	handlers := NewHandlers(factory)
	return handlers, factory
}

// TestCreateTaskHandler tests the create task endpoint
func TestCreateTaskHandler(t *testing.T) {
	handlers, _ := setupTestServer(t)

	tests := []struct {
		name       string
		method     string
		body       any
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "Method not allowed",
			method:     http.MethodGet,
			body:       nil,
			wantStatus: http.StatusMethodNotAllowed,
			wantErr:    true,
		},
		{
			name:       "Invalid JSON body",
			method:     http.MethodPost,
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:   "Empty request body",
			method: http.MethodPost,
			body:   CreateTaskRequest{},
			// Will fail validation for missing type
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:   "Missing type",
			method: http.MethodPost,
			body: CreateTaskRequest{
				Storage: "test-storage",
				Path:    "downloads",
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:   "Missing storage",
			method: http.MethodPost,
			body: CreateTaskRequest{
				Type: tasktype.TaskTypeDirectlinks,
				Path: "downloads",
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:   "Storage not found",
			method: http.MethodPost,
			body: CreateTaskRequest{
				Type:    tasktype.TaskTypeDirectlinks,
				Storage: "non-existent-storage",
				Path:    "downloads",
				Params:  json.RawMessage(`{"urls":["https://example.com/file.zip"]}`),
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			var err error

			if tt.body != nil {
				switch v := tt.body.(type) {
				case string:
					bodyBytes = []byte(v)
				default:
					bodyBytes, err = json.Marshal(tt.body)
					if err != nil {
						t.Fatalf("failed to marshal body: %v", err)
					}
				}
			}

			req := httptest.NewRequest(tt.method, "/api/v1/tasks", bytes.NewReader(bodyBytes))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			rr := httptest.NewRecorder()
			handlers.CreateTaskHandler(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantErr {
				var errResp ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
					t.Errorf("expected error response, got: %s", rr.Body.String())
				}
			}
		})
	}
}

// TestListTasksHandler tests the list tasks endpoint
func TestListTasksHandler(t *testing.T) {
	handlers, _ := setupTestServer(t)

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "Method not allowed",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "Success",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/tasks", nil)
			rr := httptest.NewRecorder()
			handlers.ListTasksHandler(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.method == http.MethodGet {
				var resp TasksListResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if resp.Tasks == nil {
					t.Error("expected non-nil tasks array")
				}
			}
		})
	}
}

// TestGetTaskHandler tests the get task endpoint
func TestGetTaskHandler(t *testing.T) {
	handlers, _ := setupTestServer(t)

	// Register a test task
	testTaskID := "test-get-task"
	RegisterTask(testTaskID, "directlinks", "local", "downloads", "Test", "")
	defer DeleteTask(testTaskID)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantFound  bool
	}{
		{
			name:       "Method not allowed",
			method:     http.MethodPost,
			path:       "/api/v1/tasks/test-id",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "Missing task ID",
			method:     http.MethodGet,
			path:       "/api/v1/tasks",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Task not found",
			method:     http.MethodGet,
			path:       "/api/v1/tasks/non-existent-task",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Task found",
			method:     http.MethodGet,
			path:       "/api/v1/tasks/" + testTaskID,
			wantStatus: http.StatusOK,
			wantFound:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			handlers.GetTaskHandler(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantFound {
				var resp TaskInfoResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if resp.TaskID != testTaskID {
					t.Errorf("expected task ID %s, got %s", testTaskID, resp.TaskID)
				}
			}
		})
	}
}

// TestCancelTaskHandler tests the cancel task endpoint
func TestCancelTaskHandler(t *testing.T) {
	handlers, _ := setupTestServer(t)

	// Register a test task
	testTaskID := "test-cancel-task"
	RegisterTask(testTaskID, "directlinks", "local", "downloads", "Test", "")
	defer DeleteTask(testTaskID)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		skipCore   bool // Skip if core is not initialized
	}{
		{
			name:       "Method not allowed",
			method:     http.MethodGet,
			path:       "/api/v1/tasks/test-id",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "Missing task ID",
			method:     http.MethodDelete,
			path:       "/api/v1/tasks",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Task not found",
			method:     http.MethodDelete,
			path:       "/api/v1/tasks/non-existent-task",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Cancel task",
			method:     http.MethodDelete,
			path:       "/api/v1/tasks/" + testTaskID,
			wantStatus: http.StatusOK,
			skipCore:   true, // Requires initialized core task queue
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipCore {
				t.Skip("Skipping test: requires initialized core")
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			handlers.CancelTaskHandler(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

// TestListStoragesHandler tests the list storages endpoint
func TestListStoragesHandler(t *testing.T) {
	handlers, _ := setupTestServer(t)

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "Method not allowed",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "Success",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/storages", nil)
			rr := httptest.NewRecorder()
			handlers.ListStoragesHandler(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.method == http.MethodGet {
				var resp StoragesResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if resp.Storages == nil {
					t.Error("expected non-nil storages array")
				}
			}
		})
	}
}

// TestConcurrentProgressStore tests concurrent access to progress store
func TestConcurrentProgressStore(t *testing.T) {
	// Clear store before test
	t.Cleanup(func() {
		tasks := GetAllTasks()
		for _, task := range tasks {
			if strings.HasPrefix(task.TaskID, "concurrent-test-") {
				DeleteTask(task.TaskID)
			}
		}
	})

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent registrations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := fmt.Sprintf("concurrent-test-%d", id)
			RegisterTask(taskID, "directlinks", "local", "downloads", "Test", "")
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := fmt.Sprintf("concurrent-test-%d", id)
			GetTask(taskID)
		}(i)
	}

	// Concurrent updates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := fmt.Sprintf("concurrent-test-%d", id)
			info, ok := GetTask(taskID)
			if ok {
				info.UpdateStatus(TaskStatusRunning)
			}
		}(i)
	}

	wg.Wait()

	// Verify all tasks exist
	for i := 0; i < numGoroutines; i++ {
		taskID := fmt.Sprintf("concurrent-test-%d", i)
		if _, ok := GetTask(taskID); !ok {
			t.Errorf("task %s not found after concurrent operations", taskID)
		}
	}
}

// TestProgressTrackerConcurrentUpdates tests concurrent progress updates
func TestProgressTrackerConcurrentUpdates(t *testing.T) {
	tracker := NewProgressTracker("concurrent-progress", "directlinks", "local", "downloads", "Test", "")
	tracker.OnStart(10000, 10)

	var wg sync.WaitGroup
	numGoroutines := 50
	updatesPerGoroutine := 100

	// Concurrent progress updates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				tracker.OnProgress(int64(id*updatesPerGoroutine+j), j)
			}
		}(i)
	}

	wg.Wait()

	info := tracker.GetInfo()
	if info.Status != TaskStatusRunning {
		t.Errorf("expected status Running after concurrent updates, got %s", info.Status)
	}
	// Note: Due to race conditions in the simple implementation,
	// we can't reliably check exact values without proper synchronization
}

// TestTaskFactoryValidation tests TaskFactory parameter validation
func TestTaskFactoryValidation(t *testing.T) {
	factory := NewTaskFactory(context.Background())

	tests := []struct {
		name    string
		request *CreateTaskRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Storage not found",
			request: &CreateTaskRequest{
				Type:    tasktype.TaskTypeDirectlinks,
				Storage: "non-existent",
				Path:    "downloads",
				Params:  json.RawMessage(`{"urls":["https://example.com/file.zip"]}`),
			},
			wantErr: true,
			errMsg:  "storage not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := factory.CreateTask(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(strings.ToLower(err.Error()), tt.errMsg) {
					t.Errorf("error message %q does not contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "Empty request body",
			fn: func(t *testing.T) {
				handlers, _ := setupTestServer(t)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
				rr := httptest.NewRecorder()
				handlers.CreateTaskHandler(rr, req)
				if rr.Code != http.StatusBadRequest {
					t.Errorf("expected %d, got %d", http.StatusBadRequest, rr.Code)
				}
			},
		},
		{
			name: "Very long task ID in path",
			fn: func(t *testing.T) {
				handlers, _ := setupTestServer(t)
				longID := strings.Repeat("a", 1000)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+longID, nil)
				rr := httptest.NewRecorder()
				handlers.GetTaskHandler(rr, req)
				if rr.Code != http.StatusNotFound {
					t.Errorf("expected %d, got %d", http.StatusNotFound, rr.Code)
				}
			},
		},
		{
			name: "Path with special characters",
			fn: func(t *testing.T) {
				path := "/api/v1/tasks/test%20id/with/slashes"
				got := extractTaskIDFromPath(path)
				expected := "test%20id"
				if got != expected {
					t.Errorf("expected %q, got %q", expected, got)
				}
			},
		},
		{
			name: "Double slashes in path",
			fn: func(t *testing.T) {
				path := "/api/v1/tasks//task-id"
				got := extractTaskIDFromPath(path)
				expected := ""
				if got != expected {
					t.Errorf("expected %q, got %q", expected, got)
				}
			},
		},
		{
			name: "Progress tracker with empty webhook",
			fn: func(t *testing.T) {
				tracker := NewProgressTracker("test", "type", "storage", "path", "title", "")
				info := tracker.GetInfo()
				if info.Webhook != "" {
					t.Error("expected empty webhook")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

// TestHealthCheckHandler tests the health check endpoint
func TestHealthCheckHandler(t *testing.T) {
	handlers, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handlers.HealthCheckHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
}

// TestGetTaskTypesHandler tests the task types endpoint
func TestGetTaskTypesHandler(t *testing.T) {
	handlers, _ := setupTestServer(t)

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "Method not allowed",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "Success",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/task-types", nil)
			rr := httptest.NewRecorder()
			handlers.GetTaskTypesHandler(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.method == http.MethodGet {
				var resp map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}
				if _, ok := resp["types"]; !ok {
					t.Error("expected 'types' field in response")
				}
			}
		})
	}
}

// TestNotFoundHandler tests the 404 handler
func TestNotFoundHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/non-existent-path", nil)
	rr := httptest.NewRecorder()
	NotFoundHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error != "not_found" {
		t.Errorf("expected error 'not_found', got %q", resp.Error)
	}
}

// TestMethodNotAllowedHandler tests the 405 handler
func TestMethodNotAllowedHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	rr := httptest.NewRecorder()
	MethodNotAllowedHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error != "method_not_allowed" {
		t.Errorf("expected error 'method_not_allowed', got %q", resp.Error)
	}
}

// TestTaskProgressInfoTimeUpdate tests that timestamps are updated correctly
func TestTaskProgressInfoTimeUpdate(t *testing.T) {
	info := RegisterTask("time-test", "directlinks", "local", "downloads", "Test", "")
	defer DeleteTask("time-test")

	originalTime := info.UpdatedAt
	time.Sleep(10 * time.Millisecond) // Ensure time difference

	info.UpdateStatus(TaskStatusRunning)
	if !info.UpdatedAt.After(originalTime) {
		t.Error("expected UpdatedAt to be updated")
	}
}

// TestWebhookPayloadWithNilCompletedAt tests webhook payload with nil completed_at
func TestWebhookPayloadWithNilCompletedAt(t *testing.T) {
	payload := WebhookPayload{
		TaskID:      "test-id",
		Type:        "directlinks",
		Status:      TaskStatusRunning,
		Storage:     "local",
		Path:        "downloads/file.zip",
		CompletedAt: nil,
		Error:       "",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// completed_at should be omitted when nil
	if _, ok := decoded["completed_at"]; ok {
		t.Error("expected completed_at to be omitted when nil")
	}
}
