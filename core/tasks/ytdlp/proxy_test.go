package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	ytdlp "github.com/lrstanley/go-ytdlp"

	"github.com/krau/SaveAny-Bot/config"
)

// fakeYtdlp writes a stub yt-dlp that records its arguments, so tests never
// touch the network.
func fakeYtdlp(t *testing.T, script string) (bin, logPath string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell stub is POSIX-only")
	}
	dir := t.TempDir()
	bin = filepath.Join(dir, "yt-dlp")
	logPath = filepath.Join(dir, "invocations.log")
	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + logPath + "\n" + script
	if err := os.WriteFile(bin, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake yt-dlp: %v", err)
	}
	return bin, logPath
}

func invocations(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read invocation log: %v", err)
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func TestProxyCandidates(t *testing.T) {
	cases := []struct {
		name    string
		proxies []string
		want    []string
	}{
		{name: "none configured runs once without a proxy", want: []string{""}},
		{name: "single proxy", proxies: []string{"http://a:1"}, want: []string{"http://a:1"}},
		{
			name:    "order is preserved so the first entry is preferred",
			proxies: []string{"http://a:1", "socks5://b:2", "http://c:3"},
			want:    []string{"http://a:1", "socks5://b:2", "http://c:3"},
		},
		{name: "blank entries dropped", proxies: []string{"", "  ", "http://a:1"}, want: []string{"http://a:1"}},
		{name: "surrounding whitespace trimmed", proxies: []string{" http://a:1 "}, want: []string{"http://a:1"}},
		{
			name:    "duplicates dropped, first position kept",
			proxies: []string{"http://a:1", "http://b:2", "http://a:1"},
			want:    []string{"http://a:1", "http://b:2"},
		},
		{name: "all entries blank falls back to a direct attempt", proxies: []string{"", " "}, want: []string{""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := proxyCandidates(config.YtdlpConfig{Proxies: tc.proxies})
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("proxyCandidates() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestRunWithProxyFailoverUsesFirstProxy(t *testing.T) {
	bin, logPath := fakeYtdlp(t, "exit 0\n")
	cfg := config.YtdlpConfig{Proxies: []string{"http://first:1", "http://second:2"}}

	if _, err := RunWithProxyFailover(t.Context(), ytdlp.New().SetExecutable(bin), cfg, "URL"); err != nil {
		t.Fatalf("RunWithProxyFailover: %v", err)
	}
	got := invocations(t, logPath)
	if len(got) != 1 || !strings.Contains(got[0], "--proxy http://first:1") {
		t.Fatalf("want one run via the first proxy, got %q", got)
	}
}

func TestRunWithProxyFailoverFallsThroughToLater(t *testing.T) {
	bin, logPath := fakeYtdlp(t, `
case "$*" in
  *--proxy\ http://alive:3*) exit 0 ;;
esac
exit 1
`)
	proxies := []string{"http://dead:1", "http://alsodead:2", "http://alive:3"}

	if _, err := RunWithProxyFailover(t.Context(), ytdlp.New().SetExecutable(bin), config.YtdlpConfig{Proxies: proxies}, "URL"); err != nil {
		t.Fatalf("want the third proxy to succeed, got %v", err)
	}
	got := invocations(t, logPath)
	if len(got) != 3 {
		t.Fatalf("want 3 runs, got %d: %q", len(got), got)
	}
	for i, p := range proxies {
		if !strings.Contains(got[i], "--proxy "+p) || strings.Count(got[i], "--proxy") != 1 {
			t.Fatalf("run %d: want exactly proxy %s, got %q", i+1, p, got[i])
		}
	}
}

func TestRunWithProxyFailoverAllFail(t *testing.T) {
	bin, logPath := fakeYtdlp(t, "echo boom >&2\nexit 1\n")
	cfg := config.YtdlpConfig{Proxies: []string{"http://a:1", "http://b:2"}}

	_, err := RunWithProxyFailover(t.Context(), ytdlp.New().SetExecutable(bin), cfg, "URL")
	if err == nil || !strings.Contains(err.Error(), "all 2 proxies failed") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want an error naming the attempts and the last stderr, got %v", err)
	}
	if n := len(invocations(t, logPath)); n != 2 {
		t.Fatalf("want 2 runs, got %d", n)
	}
}

func TestRunWithProxyFailoverNoProxyConfigured(t *testing.T) {
	bin, logPath := fakeYtdlp(t, "exit 0\n")

	if _, err := RunWithProxyFailover(t.Context(), ytdlp.New().SetExecutable(bin), config.YtdlpConfig{}, "URL"); err != nil {
		t.Fatalf("RunWithProxyFailover: %v", err)
	}
	got := invocations(t, logPath)
	if len(got) != 1 || strings.Contains(got[0], "--proxy") {
		t.Fatalf("want one direct run, got %q", got)
	}
}

func TestRunWithProxyFailoverSingleFailureKeepsError(t *testing.T) {
	bin, _ := fakeYtdlp(t, "echo boom >&2\nexit 1\n")

	_, err := RunWithProxyFailover(t.Context(), ytdlp.New().SetExecutable(bin), config.YtdlpConfig{}, "URL")
	if err == nil || strings.Contains(err.Error(), "proxies failed") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want the plain yt-dlp error, got %v", err)
	}
}

func TestRunWithProxyFailoverStopsOnCancel(t *testing.T) {
	bin, logPath := fakeYtdlp(t, "exit 1\n")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	cfg := config.YtdlpConfig{Proxies: []string{"http://a:1", "http://b:2", "http://c:3"}}

	if _, err := RunWithProxyFailover(ctx, ytdlp.New().SetExecutable(bin), cfg, "URL"); err == nil {
		t.Fatal("want an error from a cancelled context")
	}
	if n := len(invocations(t, logPath)); n > 1 {
		t.Fatalf("cancelled run kept trying proxies: %d runs", n)
	}
}

// yt-dlp echoes the proxy URL in some connection errors; the password must not
// reach task errors, which are stored and sent to webhooks.
func TestRunWithProxyFailoverRedactsCredentialsInErrors(t *testing.T) {
	bin, _ := fakeYtdlp(t, "echo \"ERROR: Unable to connect to proxy: $*\" >&2\nexit 1\n")
	cfg := config.YtdlpConfig{Proxies: []string{"socks5://user:s3cret@host:1080"}}

	_, err := RunWithProxyFailover(t.Context(), ytdlp.New().SetExecutable(bin), cfg, "URL")
	if err == nil {
		t.Fatal("want an error")
	}
	if strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("error leaked the proxy password: %v", err)
	}
}

func TestRedactProxy(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "direct"},
		{"http://host:8080", "http://host:8080"},
		{"socks5://user:s3cret@host:1080", "socks5://user:xxxxx@host:1080"},
		{"http://user@host:8080", "http://user@host:8080"},
		{"not a url at all", "<proxy>"},
	}
	for _, tc := range cases {
		if got := redactProxy(tc.in); got != tc.want {
			t.Errorf("redactProxy(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCollectDownloadedFilesSkipsTempFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"video.mp4", "video.mp4.part", "video.info.ytdl", "video.temp.mp4.temp", "partial.mp3"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := collectDownloadedFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"partial.mp3", "video.mp4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectDownloadedFiles() = %q, want %q", got, want)
	}
}
