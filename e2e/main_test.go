//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var (
	serverURL string
	dataDir   string
	repoRoot  string
)

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	var err error
	repoRoot, err = filepath.Abs("..")
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolve repo root:", err)
		return 1
	}
	binDir, err := os.MkdirTemp("", "saveany-e2e-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create binary directory:", err)
		return 1
	}
	defer os.RemoveAll(binDir)
	dataDir, err = os.MkdirTemp("", "saveany-e2e-data-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create data directory:", err)
		return 1
	}
	defer os.RemoveAll(dataDir)

	bin := filepath.Join(binDir, "saveany-bot")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = repoRoot
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "build test binary:", err)
		return 1
	}

	port := freePort()
	serverURL = "http://127.0.0.1:" + port
	logFile, err := os.CreateTemp("", "saveany-e2e-*.log")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create log file:", err)
		return 1
	}
	defer func() {
		name := logFile.Name()
		_ = logFile.Close()
		if os.Getenv("E2E_KEEP_LOG") == "" {
			_ = os.Remove(name)
		} else {
			fmt.Fprintln(os.Stderr, "server log:", name)
		}
	}()

	cmd := exec.Command(bin,
		"testenv",
		"--data-dir", dataDir,
		"--listen", "127.0.0.1:"+port,
		"--admin-password", "test-admin",
		"--storage-name", "e2e-local",
		"--user-chat-id", "10001",
	)
	cmd.Dir = dataDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(), "SAVEANY_ADMIN_PASSWORD=test-admin")
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "start test environment:", err)
		return 1
	}

	stopped := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(stopped)
	}()
	waitCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := waitForHealth(waitCtx); err != nil {
		select {
		case <-stopped:
		default:
			_ = cmd.Process.Kill()
		}
		_ = logFile.Sync()
		data, _ := os.ReadFile(logFile.Name())
		fmt.Fprintf(os.Stderr, "test environment did not become ready: %v\n%s\n", err, data)
		return 1
	}

	code := m.Run()
	_ = cmd.Process.Signal(os.Interrupt)
	select {
	case <-stopped:
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		<-stopped
	}
	return code
}

func freePort() string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	return port
}

func waitForHealth(ctx context.Context) error {
	client := &http.Client{Timeout: 2 * time.Second}
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/health", nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

var shared *apiClient

// sharedClient logs in once for tests that only need an authenticated session.
// The login limiter allows a small burst per address, so creating a fresh
// session in every test would exhaust it.
func sharedClient(t *testing.T) *apiClient {
	t.Helper()
	if shared == nil {
		shared = newClient(t)
	}
	c := *shared
	c.t = t
	return &c
}

type apiClient struct {
	t      *testing.T
	http   *http.Client
	csrf   string
	base   string
	authed bool
}

func newClient(t *testing.T) *apiClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	c := &apiClient{t: t, base: serverURL, http: &http.Client{Jar: jar, Timeout: 10 * time.Second}}
	status, body := c.request(http.MethodPost, "/api/v1/auth/login", map[string]string{"password": "test-admin"}, false)
	if status != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", status, body)
	}
	var login struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(body, &login); err != nil || login.CSRF == "" {
		t.Fatalf("invalid login response: %s", body)
	}
	c.csrf = login.CSRF
	c.authed = true
	return c
}

func (c *apiClient) request(method, path string, body any, withCSRF bool) (int, []byte) {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if withCSRF {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatalf("read %s %s response: %v", method, path, err)
	}
	return resp.StatusCode, data
}

func (c *apiClient) do(method, path string, body any, withCSRF bool, out any) int {
	c.t.Helper()
	status, data := c.request(method, path, body, withCSRF)
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			c.t.Fatalf("%s %s returned invalid JSON: %v\n%s", method, path, err, data)
		}
	}
	return status
}
