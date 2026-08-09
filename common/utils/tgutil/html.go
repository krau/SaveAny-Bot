package tgutil

import (
	stdhtml "html"

	"github.com/gotd/td/telegram/message/entity"
	messagehtml "github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
)

// EscapeHTMLTemplateData returns a copy of data with string values escaped for
// interpolation into Telegram HTML templates.
func EscapeHTMLTemplateData(data map[string]any) map[string]any {
	escaped := make(map[string]any, len(data))
	for key, value := range data {
		if text, ok := value.(string); ok {
			escaped[key] = stdhtml.EscapeString(text)
			continue
		}
		escaped[key] = value
	}
	return escaped
}

// RenderHTML renders Telegram-compatible HTML into plain text and message
// entities.
func RenderHTML(markup string) (string, []tg.MessageEntityClass, error) {
	var builder entity.Builder
	if err := styling.Perform(&builder, messagehtml.String(nil, markup)); err != nil {
		return "", nil, err
	}
	text, entities := builder.Complete()
	return text, entities, nil
}
