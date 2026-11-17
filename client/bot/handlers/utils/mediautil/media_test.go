package mediautil

import (
	"context"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/database"
)

func TestFilenameForSanitizesOriginalName(t *testing.T) {
	t.Parallel()

	message := testDocumentMessage()
	if got := FilenameFor(context.Background(), &database.User{}, message); got != "file.txt" {
		t.Fatalf("FilenameFor() = %q, want file.txt", got)
	}
}

func TestBuildFilenameTemplateDataSanitizesOriginalName(t *testing.T) {
	t.Parallel()

	message := testDocumentMessage()
	data := BuildFilenameTemplateData(message)
	if got := data["origname"]; got != "file.txt" {
		t.Fatalf("origname = %q, want file.txt", got)
	}
}

func testDocumentMessage() *tg.Message {
	return &tg.Message{
		ID: 1,
		Media: &tg.MessageMediaDocument{
			Document: &tg.Document{
				ID: 1,
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeFilename{FileName: `linked\file.txt`},
				},
			},
		},
	}
}
