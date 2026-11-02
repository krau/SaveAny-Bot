package telegram

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateSplitZip(t *testing.T) {
	content := bytes.Repeat([]byte("SaveAny-Bot split test\n"), 2*1024*1024/22+1)
	content = content[:2*1024*1024+123]
	tests := []struct {
		name     string
		partSize int64
	}{
		{name: "1MiB", partSize: 1024 * 1024},
		{name: "512KiB", partSize: 512 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "split")
			if err := CreateSplitZip(t.Context(), bytes.NewReader(content), int64(len(content)), "testfile.dat", output, tt.partSize); err != nil {
				t.Fatalf("CreateSplitZip failed: %v", err)
			}
			matched, err := filepath.Glob(output + ".z*")
			if err != nil {
				t.Fatalf("failed to glob split files: %v", err)
			}
			if len(matched) == 0 {
				t.Fatal("no split files found")
			}
			for _, name := range matched {
				info, err := os.Stat(name)
				if err != nil {
					t.Fatalf("failed to stat %s: %v", name, err)
				}
				if info.Size() > tt.partSize {
					t.Errorf("file %s exceeds part size: %d > %d", name, info.Size(), tt.partSize)
				}
			}
		})
	}
}
