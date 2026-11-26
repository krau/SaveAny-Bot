package api

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReceiveUploadReadIdleDeadline(t *testing.T) {
	for _, stalled := range []bool{false, true} {
		t.Run(map[bool]string{false: "slow active sender", true: "stalled sender"}[stalled], func(t *testing.T) {
			payload := filepath.Join(t.TempDir(), "payload")
			result := make(chan error, 1)
			idle := 500 * time.Millisecond
			if stalled {
				idle = 100 * time.Millisecond
			}
			server := httptest.NewUnstartedServer(loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := receiveUpload(w, r, payload, 100, idle)
				result <- err
			})))
			server.Config.ReadTimeout = 100 * time.Millisecond
			server.Start()
			defer server.Close()

			conn, err := net.Dial("tcp", server.Listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			if _, err := fmt.Fprint(conn, "POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 4\r\nConnection: close\r\n\r\na"); err != nil {
				t.Fatal(err)
			}
			if !stalled {
				for _, ch := range []string{"b", "c", "d"} {
					time.Sleep(200 * time.Millisecond)
					if _, err := fmt.Fprint(conn, ch); err != nil {
						t.Fatal(err)
					}
				}
			}
			select {
			case err = <-result:
			case <-time.After(3 * time.Second):
				t.Fatal("upload handler did not finish")
			}
			if stalled {
				var netErr net.Error
				if !errors.As(err, &netErr) || !netErr.Timeout() {
					t.Fatalf("stalled upload error = %v, want read timeout", err)
				}
				if _, err := os.Stat(payload); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("stalled upload published a payload: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("active upload timed out: %v", err)
			}
			if data, err := os.ReadFile(payload); err != nil || string(data) != "abcd" {
				t.Fatalf("staged payload = %q, %v", data, err)
			}
		})
	}
}
