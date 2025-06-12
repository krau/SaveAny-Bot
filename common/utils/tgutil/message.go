package tgutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
)

func GenFileNameFromMessage(message tg.Message) string {
	text := strings.TrimSpace(message.GetMessage())
	if text == "" {
		return ""
	}
	filename := func() string {
		tags := strutil.ExtractTagsFromText(text)
		if len(tags) > 0 {
			return fmt.Sprintf("%s_%s", strings.Join(tags, "_"), strconv.Itoa(message.GetID()))
		}
		text = strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
				return '_'
			}
			return r
		}, text)
		runes := []rune(text)
		return string(runes[:min(128, len(runes))])
	}()
	ext := func(media tg.MessageMediaClass) string {
		switch media := media.(type) {
		case *tg.MessageMediaDocument:
			doc, ok := media.Document.AsNotEmpty()
			if !ok {
				return ""
			}
			ext := mimetype.Lookup(doc.MimeType).Extension()
			if ext == "" {
				return ""
			}
			return ext
		case *tg.MessageMediaPhoto:
			return ".jpg"
		}
		return ""
	}(message.Media)
	return filename + ext
}
