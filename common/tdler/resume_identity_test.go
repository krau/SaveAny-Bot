package tdler

import (
	"testing"

	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

func TestResumeBitmapFileIdentity(t *testing.T) {
	a := tfile.FileIdentity{Kind: "document", ID: 1, AccessHash: 2}
	b := tfile.FileIdentity{Kind: "document", ID: 3, AccessHash: 2}
	bm := newResumeBitmap(2*1024*1024, a)
	if !bm.matches(a) {
		t.Fatal("bitmap did not match its source file")
	}
	if bm.matches(b) {
		t.Fatal("bitmap matched a different source file")
	}
	legacy := &resumeBitmap{PartSize: 1024, Size: 1}
	if !legacy.matches(b) {
		t.Fatal("legacy bitmap without identity should be accepted")
	}
}
