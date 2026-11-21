package storage_test

import (
	"context"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/storage"
	"golang.org/x/net/webdav"
)

func TestWriteMetadataSidecarWithMediaContentLength(t *testing.T) {
	fs := webdav.NewMemFS()
	server := httptest.NewServer(&webdav.Handler{Prefix: "/dav", FileSystem: fs, LockSystem: webdav.NewMemLS()})
	t.Cleanup(server.Close)
	st, err := storage.NewStorage(t.Context(), &storcfg.WebdavStorageConfig{
		URL:      server.URL + "/dav",
		BasePath: "/media",
		Username: "user",
		Password: "pass",
	})
	if err != nil {
		t.Fatal(err)
	}

	const data = `{"caption":"photo"}`
	ctx := context.WithValue(t.Context(), ctxkey.ContentLength, int64(8192))
	if err := storage.WriteMetadataSidecar(ctx, st, "album/photo.jpg", []byte(data)); err != nil {
		t.Fatal(err)
	}
	f, err := fs.OpenFile(t.Context(), "/media/album/photo.jpg.meta.json", os.O_RDONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := io.ReadAll(f)
	if err != nil || string(got) != data {
		t.Fatalf("sidecar contents = %q, %v; want %q, nil", got, err, data)
	}
}
