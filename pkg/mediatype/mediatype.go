// Package mediatype classifies Telegram media into the coarse types used by
// the instance-wide media filter.
package mediatype

import (
	"fmt"
	"path"
	"strings"

	"github.com/gotd/td/tg"
)

// Type is a user-facing media category. It is intentionally coarser than
// Telegram's document attributes: every Telegram document falls into one of
// video, audio, image, or document.
type Type string

const (
	Image    Type = "image"
	Video    Type = "video"
	Audio    Type = "audio"
	Document Type = "document"
)

var allTypes = []Type{Image, Video, Audio, Document}

// Parse normalizes a configured media type. "photo" is accepted as an alias
// for "image".
func Parse(value string) (Type, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image", "photo":
		return Image, true
	case "video":
		return Video, true
	case "audio":
		return Audio, true
	case "document":
		return Document, true
	default:
		return "", false
	}
}

// NormalizeAllowed validates and canonicalizes an allowlist. Empty values are
// ignored; an empty result means that no filter is active.
func NormalizeAllowed(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := make(map[Type]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		typ, ok := Parse(value)
		if !ok {
			return nil, fmt.Errorf("invalid media type %q (allowed: image, video, audio, document)", value)
		}
		if _, exists := seen[typ]; exists {
			continue
		}
		seen[typ] = struct{}{}
		out = append(out, string(typ))
	}
	return out, nil
}

// Classify returns the category of a Telegram media object. It reports false
// for unsupported media and empty documents.
func Classify(media tg.MessageMediaClass) (Type, bool) {
	switch m := media.(type) {
	case *tg.MessageMediaPhoto:
		return Image, true
	case *tg.MessageMediaDocument:
		document, ok := m.Document.AsNotEmpty()
		if !ok {
			return "", false
		}
		return classifyDocument(document), true
	default:
		return "", false
	}
}

func classifyDocument(document *tg.Document) Type {
	var hasAnimated, hasSticker, hasVideo, hasAudio, hasImage bool
	for _, attribute := range document.Attributes {
		switch attribute.(type) {
		case *tg.DocumentAttributeAnimated:
			hasAnimated = true
		case *tg.DocumentAttributeSticker:
			hasSticker = true
		case *tg.DocumentAttributeVideo:
			hasVideo = true
		case *tg.DocumentAttributeAudio:
			hasAudio = true
		case *tg.DocumentAttributeImageSize:
			hasImage = true
		}
	}

	mimeType := strings.ToLower(document.MimeType)
	switch {
	case hasSticker || hasAnimated:
		return Image
	case hasVideo:
		return Video
	case hasAudio:
		return Audio
	case hasImage:
		return Image
	case strings.HasPrefix(mimeType, "image/"):
		return Image
	case strings.HasPrefix(mimeType, "video/"):
		return Video
	case strings.HasPrefix(mimeType, "audio/"):
		return Audio
	}

	switch extension(documentFileName(document)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".avif", ".heic":
		return Image
	case ".mp4", ".mkv", ".ts", ".avi", ".flv", ".mov", ".webm", ".wmv", ".rmvb", ".m2ts":
		return Video
	case ".mp3", ".flac", ".wav", ".aac", ".m4a", ".ogg", ".opus":
		return Audio
	default:
		return Document
	}
}

func documentFileName(document *tg.Document) string {
	for _, attribute := range document.Attributes {
		if name, ok := attribute.(*tg.DocumentAttributeFilename); ok {
			return name.GetFileName()
		}
	}
	return ""
}

func extension(name string) string {
	return strings.ToLower(path.Ext(name))
}

// All returns the supported filter categories.
func All() []Type {
	return append([]Type(nil), allTypes...)
}
