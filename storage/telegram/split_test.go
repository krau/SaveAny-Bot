package telegram

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateSplitZip(t *testing.T) {
	input := "tests/testfile.dat"
	file, err := os.Open(input)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}
	defer file.Close()
	fileName := filepath.Base(input)
	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}
	fileSize := fileInfo.Size()

	tests := []struct {
		partSize int64
		output   string
	}{
		{partSize: int64(1024 * 1024 * 500), output: "tests/split_test_output_500MB"},
		{partSize: int64(1024 * 1024 * 100), output: "tests/split_test_output_100MB"},
	}

	for _, tt := range tests {
		err = CreateSplitZip(t.Context(), file, fileSize, fileName, tt.output, tt.partSize)
		if err != nil {
			t.Fatalf("CreateSplitZip failed: %v", err)
		}
		matched, err := filepath.Glob(tt.output + ".z*")
		if err != nil {
			t.Fatalf("failed to glob split files: %v", err)
		}
		if len(matched) == 0 {
			t.Fatalf("no split files found")
		}
		t.Logf("Created %d split files", len(matched))
		for _, f := range matched {
			info, err := os.Stat(f)
			if err != nil {
				t.Fatalf("failed to stat file %s: %v", f, err)
			}
			if info.Size() > tt.partSize {
				t.Errorf("file %s exceeds part size: %d > %d", f, info.Size(), tt.partSize)
			}
			t.Logf(" - %s (%d bytes)", f, info.Size())
		}
	}
}
