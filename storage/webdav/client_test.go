package webdav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/webdav"
)

func setupWebDAVServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "webdav_test")
	if err != nil {
		t.Fatalf("mk temp dir failed: %v", err)
	}

	handler := &webdav.Handler{
		Prefix:     "/",
		FileSystem: webdav.Dir(tempDir),
		LockSystem: webdav.NewMemLS(),
	}

	server := httptest.NewServer(handler)
	return server, tempDir
}

func TestMkDirAndExists(t *testing.T) {
	server, tempDir := setupWebDAVServer(t)
	defer os.RemoveAll(tempDir)
	defer server.Close()

	client := NewClient(server.URL, "", "", nil)
	ctx := context.Background()

	testpaths := []string{"testdir", "testdir/subdir", "testdir/子目录", "/testdir/测试路径/测试路径2"}
	for _, p := range testpaths {
		exists, err := client.Exists(ctx, p)
		if err != nil {
			t.Fatalf("Call Exists Err: %v", err)
		}
		if exists {
			t.Fatalf("Dir should not exist")
		}

		if err := client.MkDir(ctx, p); err != nil {
			t.Fatalf("Call MkDir Err: %v", err)
		}

		exists, err = client.Exists(ctx, p)
		if err != nil {
			t.Fatalf("Call Exists Err: %v", err)
		}
		if !exists {
			t.Fatalf("Dir should exist")
		}
	}

}

func TestSpecialCharsInPaths(t *testing.T) {
	fs := webdav.NewMemFS()
	server := httptest.NewServer(&webdav.Handler{Prefix: "/dav", FileSystem: fs, LockSystem: webdav.NewMemLS()})
	t.Cleanup(server.Close)
	client := NewClient(server.URL+"/dav", "", "", server.Client())

	for _, file := range []string{"one\ntwo.txt", "one#two.txt", "one?two.txt", "one%20two.txt"} {
		t.Run(file, func(t *testing.T) {
			if err := client.WriteFile(t.Context(), file, strings.NewReader("content")); err != nil {
				t.Fatal(err)
			}
			if exists, err := client.Exists(t.Context(), file); err != nil || !exists {
				t.Fatalf("Exists(%q) = %v, %v; want true, nil", file, exists, err)
			}
		})
	}

	t.Run("mkdir", func(t *testing.T) {
		const dir = "/album#1/subdir?%20"
		if err := client.MkDir(t.Context(), dir); err != nil {
			t.Fatal(err)
		}
		if info, err := fs.Stat(t.Context(), dir); err != nil || !info.IsDir() {
			t.Fatalf("Stat(%q) = %v, %v; want a directory", dir, info, err)
		}
	})
}

func TestExistsKeepsEscapedBaseURLAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const want = "/dav%2Fmount/file%23name?key=a%2Fb"
		if r.RequestURI != want {
			t.Errorf("request URI = %q, want %q", r.RequestURI, want)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusMultiStatus)
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.URL+"/dav%2Fmount?key=a%2Fb", "", "", server.Client())
	if exists, err := client.Exists(t.Context(), "file#name"); err != nil || !exists {
		t.Fatalf("Exists() = %v, %v; want true, nil", exists, err)
	}
}

func TestWriteFile(t *testing.T) {
	server, tempDir := setupWebDAVServer(t)
	defer os.RemoveAll(tempDir)
	defer server.Close()

	client := NewClient(server.URL, "", "", nil)
	ctx := context.Background()

	testCases := []struct {
		remotePath string
		content    string
	}{
		{
			remotePath: "hello.txt",
			content:    "Hello webdav",
		},
		{
			remotePath: "//nested/dir/test.txt",
			content:    "Nested file",
		},
		{
			remotePath: "nested/dir/test.txt",
			content:    "Nested file",
		},
		{
			remotePath: "empty.txt",
			content:    "",
		},
		{
			remotePath: "unicode.txt",
			content:    "测试",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.remotePath, func(t *testing.T) {
			dir := path.Dir(tc.remotePath)
			if dir != "." {
				if err := client.MkDir(ctx, dir); err != nil {
					t.Fatalf("创建目录 %s 失败: %v", dir, err)
				}
			}

			if err := client.WriteFile(ctx, tc.remotePath, strings.NewReader(tc.content)); err != nil {
				t.Fatalf("写入文件 %s 失败: %v", tc.remotePath, err)
			}

			localPath := filepath.Join(tempDir, tc.remotePath)
			data, err := os.ReadFile(localPath)
			if err != nil {
				t.Fatalf("读取文件 %s 失败: %v", localPath, err)
			}
			if string(data) != tc.content {
				t.Fatalf("文件内容不匹配: got %s, want %s", string(data), tc.content)
			}

			appended := tc.content + " Overwritten."
			if err := client.WriteFile(ctx, tc.remotePath, strings.NewReader(appended)); err != nil {
				t.Fatalf("覆盖写入文件 %s 失败: %v", tc.remotePath, err)
			}
			data, err = os.ReadFile(localPath)
			if err != nil {
				t.Fatalf("读取覆盖后的文件 %s 失败: %v", localPath, err)
			}
			if string(data) != appended {
				t.Fatalf("文件覆盖后的内容不匹配: got %s, want %s", string(data), appended)
			}
		})
	}
}
