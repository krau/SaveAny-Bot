package tfile

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestFromMediaSanitizesDocumentFilename(t *testing.T) {
	t.Parallel()

	media := &tg.MessageMediaDocument{
		Document: &tg.Document{
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeFilename{FileName: "linked/file.txt"},
			},
		},
	}
	file, err := FromMedia(media, nil)
	if err != nil {
		t.Fatalf("FromMedia: %v", err)
	}
	if got := file.Name(); got != "file.txt" {
		t.Fatalf("file.Name() = %q, want file.txt", got)
	}
}
