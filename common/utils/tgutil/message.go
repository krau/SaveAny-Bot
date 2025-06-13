package tgutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/ext"
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

func BuildCancelButton(taskID string) tg.KeyboardButtonClass {
	return &tg.KeyboardButtonCallback{
		Text: "取消任务",
		Data: fmt.Appendf(nil, "cancel %s", taskID),
	}
}

func InputMessageClassSliceFromRange(min, max int) []tg.InputMessageClass {
	if min == max {
		return []tg.InputMessageClass{
			&tg.InputMessageID{
				ID: min,
			},
		}
	}
	result := make([]tg.InputMessageClass, 0, max-min+1)
	for i := min; i <= max; i++ {
		result = append(result, &tg.InputMessageID{
			ID: i,
		})
	}
	return result
}

func GetMessages(ctx *ext.Context, chatID int64, minId, maxId int) ([]*tg.Message, error) {
	// TODO: cache
	result := make([]*tg.Message, 0, maxId-minId+1)
	for i := minId; i <= maxId; i += 100 {
		msgs, err := ctx.GetMessages(chatID, InputMessageClassSliceFromRange(i, min(i+100, maxId)))
		if err != nil {
			return nil, err
		}
		if len(msgs) == 0 {
			continue
		}
		for _, msg := range msgs {
			if msg == nil {
				continue
			}
			tgMessage, ok := msg.(*tg.Message)
			if !ok {
				continue
			}
			if tgMessage.GetID() < minId || tgMessage.GetID() > maxId {
				continue
			}
			result = append(result, tgMessage)
		}
	}
	return result, nil
}
