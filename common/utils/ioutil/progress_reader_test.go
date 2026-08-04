package ioutil

import (
	"bytes"
	"io"
	"testing"
)

func TestProgressReadSeekerTracksReads(t *testing.T) {
	var gotRead, gotTotal int64
	reader := NewProgressReader(bytes.NewReader([]byte("abcdef")), 6, func(read, total int64) {
		gotRead = read
		gotTotal = total
	})

	buffer := make([]byte, 4)
	if _, err := io.ReadFull(reader, buffer); err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if gotRead != 4 || gotTotal != 6 {
		t.Fatalf("progress = %d/%d, want 4/6", gotRead, gotTotal)
	}
	if reader.BytesRead() != 4 {
		t.Fatalf("BytesRead() = %d, want 4", reader.BytesRead())
	}
}

func TestProgressReadSeekerResetsPositionOnSeek(t *testing.T) {
	reader := NewProgressReader(bytes.NewReader([]byte("abcdef")), 6, nil)
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(reader, buffer); err != nil {
		t.Fatalf("read failed: %v", err)
	}

	position, err := reader.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("seek failed: %v", err)
	}
	if position != 0 || reader.BytesRead() != 0 {
		t.Fatalf("position after seek = %d (tracked %d), want 0", position, reader.BytesRead())
	}

	if _, err := io.ReadFull(reader, buffer[:2]); err != nil {
		t.Fatalf("read after seek failed: %v", err)
	}
	if reader.BytesRead() != 2 {
		t.Fatalf("BytesRead() after seek and read = %d, want 2", reader.BytesRead())
	}
}
