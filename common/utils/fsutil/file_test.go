package fsutil_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
)

func TestCloseAndRemove(t *testing.T) {
	tests := []struct {
		name     string
		preClose bool
	}{
		{name: "open file"},
		{name: "already closed file", preClose: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(t.TempDir(), "cache-file")
			file, err := fsutil.CreateFile(filePath)
			if err != nil {
				t.Fatalf("CreateFile() failed: %v", err)
			}
			if tt.preClose {
				if err := file.Close(); err != nil {
					t.Fatalf("Close() failed: %v", err)
				}
			}

			if err := file.CloseAndRemove(); err != nil {
				t.Fatalf("CloseAndRemove() failed: %v", err)
			}
			if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("cache file still exists after CloseAndRemove(): %v", err)
			}
		})
	}
}
