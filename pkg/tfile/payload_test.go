package tfile

import (
	"reflect"
	"testing"

	"github.com/gotd/td/tg"
)

func TestFilePayloadDocumentRoundTrip(t *testing.T) {
	file := NewTGFile(
		&tg.InputDocumentFileLocation{
			ID:            6287403840090150101,
			AccessHash:    -8452541528324991878,
			FileReference: []byte{0x02, 0x0e, 0x80, 0xd6},
			ThumbSize:     "",
		},
		nil,
		4194304000,
		"常轨脱离Creative凸.7z.001",
	)
	p, ok := FilePayloadOf(file)
	if !ok {
		t.Fatalf("FilePayloadOf failed")
	}
	rebuilt := FileFromPayload(p, nil)
	if !reflect.DeepEqual(rebuilt.Location(), file.Location()) {
		t.Fatalf("location mismatch:\n got %#v\nwant %#v", rebuilt.Location(), file.Location())
	}
	if rebuilt.Size() != file.Size() || rebuilt.Name() != file.Name() {
		t.Fatalf("size/name mismatch: got %d %q, want %d %q", rebuilt.Size(), rebuilt.Name(), file.Size(), file.Name())
	}
	if p.Kind != "document" {
		t.Fatalf("kind = %q, want document", p.Kind)
	}
}

func TestFilePayloadPhotoRoundTrip(t *testing.T) {
	file := NewTGFile(
		&tg.InputPhotoFileLocation{
			ID:            123,
			AccessHash:    456,
			FileReference: []byte{0xaa, 0xbb},
			ThumbSize:     "y",
		},
		nil,
		0,
		"photo_123.png",
	)
	p, ok := FilePayloadOf(file)
	if !ok {
		t.Fatalf("FilePayloadOf failed")
	}
	if p.Kind != "photo" {
		t.Fatalf("kind = %q, want photo", p.Kind)
	}
	rebuilt := FileFromPayload(p, nil)
	if !reflect.DeepEqual(rebuilt.Location(), file.Location()) {
		t.Fatalf("location mismatch:\n got %#v\nwant %#v", rebuilt.Location(), file.Location())
	}
}
