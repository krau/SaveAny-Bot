//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestGracefulRestartResumesTask(t *testing.T) {
	restartResume(t, os.Interrupt)
}

func TestSIGTERMRestartResumesTask(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM process test is POSIX-only")
	}
	restartResume(t, syscall.SIGTERM)
}

// restartResume kills a real testenv process and starts a new process against
// the same data directory.
func restartResume(t *testing.T, sig os.Signal) {
	dir := t.TempDir()
	port := freePort()
	base := "http://127.0.0.1:" + port

	content := bytes.Repeat([]byte("restart-resume\n"), (16*1024*1024)/15+1)[:16*1024*1024]
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/slow.bin" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		for offset := 0; offset < len(content); offset += 256 * 1024 {
			end := offset + 256*1024
			if end > len(content) {
				end = len(content)
			}
			if _, err := w.Write(content[offset:end]); err != nil {
				return
			}
			w.(http.Flusher).Flush()
			time.Sleep(20 * time.Millisecond)
		}
	}))
	defer source.Close()

	env := startTestEnv(t, dir, port)
	client := newClientAt(t, base)
	var created struct {
		TaskID string `json:"task_id"`
	}
	if status := client.do(http.MethodPost, "/api/v1/tasks", map[string]any{
		"type":    "directlinks",
		"storage": "test-local",
		"path":    "resume",
		"params":  map[string]any{"urls": []string{source.URL + "/slow.bin"}},
	}, true, &created); status != http.StatusCreated || created.TaskID == "" {
		t.Fatalf("create task = %d %+v", status, created)
	}
	waitTaskStatus(t, client, created.TaskID, "running", 15*time.Second)

	env.signal(t, sig, 15*time.Second)

	env = startTestEnv(t, dir, port)
	defer env.signal(t, os.Interrupt, 10*time.Second)
	client = newClientAt(t, base)
	info := waitTaskStatus(t, client, created.TaskID, "completed", 30*time.Second)
	if info.Status != "completed" {
		t.Fatalf("task status = %q, items = %+v", info.Status, info.Items)
	}
	got, err := os.ReadFile(filepath.Join(dir, "downloads", "resume", "slow.bin"))
	if err != nil {
		t.Fatalf("read resumed file: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("resumed file size = %d, want %d", len(got), len(content))
	}
}

type testEnvProcess struct {
	dir     string
	port    string
	base    string
	cmd     *exec.Cmd
	stopped chan struct{}
	log     *os.File
}

func startTestEnv(t *testing.T, dir, port string) *testEnvProcess {
	t.Helper()
	logPath := filepath.Join(dir, "testenv.log")
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open testenv log: %v", err)
	}
	cmd := exec.Command(testBinary,
		"testenv",
		"--data-dir", dir,
		"--keep",
		"--listen", "127.0.0.1:"+port,
		"--admin-password", "test-admin",
		"--storage-name", "test-local",
		"--user-chat-id", "10001",
	)
	cmd.Dir = dir
	cmd.Stdout = log
	cmd.Stderr = log
	cmd.Env = append(os.Environ(), "SAVEANY_ADMIN_PASSWORD=test-admin")
	if err := cmd.Start(); err != nil {
		log.Close()
		t.Fatalf("start testenv: %v", err)
	}
	env := &testEnvProcess{dir: dir, port: port, base: "http://127.0.0.1:" + port, cmd: cmd, stopped: make(chan struct{}), log: log}
	go func() {
		_ = cmd.Wait()
		close(env.stopped)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := waitForHealthAt(ctx, env.base); err != nil {
		env.signal(t, os.Interrupt, 5*time.Second)
		t.Fatalf("testenv did not become ready: %v", err)
	}
	t.Cleanup(func() {
		select {
		case <-env.stopped:
		default:
			_ = env.cmd.Process.Kill()
			<-env.stopped
		}
		_ = env.log.Close()
	})
	return env
}

func (e *testEnvProcess) signal(t *testing.T, sig os.Signal, timeout time.Duration) {
	t.Helper()
	if err := e.cmd.Process.Signal(sig); err != nil {
		t.Fatalf("signal testenv: %v", err)
	}
	select {
	case <-e.stopped:
	case <-time.After(timeout):
		_ = e.cmd.Process.Kill()
		<-e.stopped
		t.Fatalf("testenv did not stop after %s", sig)
	}
}

func waitTaskStatus(t *testing.T, client *apiClient, id, want string, timeout time.Duration) taskInfo {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		var info taskInfo
		if status := client.do(http.MethodGet, "/api/v1/tasks/"+id, nil, false, &info); status != http.StatusOK {
			t.Fatalf("get task %s = %d", id, status)
		}
		if info.Status == want {
			return info
		}
		if want == "running" && (info.Status == "completed" || info.Status == "failed" || info.Status == "cancelled") {
			t.Fatalf("task %s finished before interruption: %s", id, info.Status)
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %s = %q, want %q", id, info.Status, want)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
