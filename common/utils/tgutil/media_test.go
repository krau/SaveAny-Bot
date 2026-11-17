package tgutil

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestTelegramFilenameIsReducedToOneComponent(t *testing.T) {
	t.Parallel()

	media := &tg.MessageMediaDocument{
		Document: &tg.Document{
			ID: 1,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeFilename{FileName: `..\linked\file.txt`},
			},
		},
	}
	if got := GetOriginalFileName(media); got != "file.txt" {
		t.Fatalf("GetOriginalFileName() = %q, want file.txt", got)
	}

	message := tg.Message{ID: 1, Message: "///", Media: media}
	if got := GenFileNameFromMessage(message); got != "file.txt" {
		t.Fatalf("GenFileNameFromMessage() = %q, want file.txt", got)
	}
}
