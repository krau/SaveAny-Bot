package tgutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/mathutil"
	"github.com/duke-git/lancet/v2/slice"
	lcstrutil "github.com/duke-git/lancet/v2/strutil"
	"github.com/duke-git/lancet/v2/validator"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/rs/xid"
)

func GenFileNameFromMessage(message tg.Message) string {
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
	text := strings.TrimSpace(message.GetMessage())
	if text == "" {
		return fmt.Sprintf("%d_%s%s", message.GetID(), xid.New().String(), ext)
	}
	filename := func() string {
		tags := strutil.ExtractTagsFromText(text)
		if len(tags) > 0 {
			tagStrRunes := make([]rune, 0, 64)
			for i, tag := range tags {
				if i > 0 {
					tagStrRunes = append(tagStrRunes, '_')
				}
				tagStrRunes = append(tagStrRunes, []rune(tag)...)
				if len(tagStrRunes) >= 64 {
					break
				}
			}
			tagStr := string(tagStrRunes)
			return fmt.Sprintf("%s_%s", tagStr, strconv.Itoa(message.GetID()))
		}
		text = lcstrutil.Substring(strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7F {
				return '_'
			}
			switch r {
			// invalid characters
			case '/', '\\',
				':', '*', '?', '"', '<', '>', '|':
				return '_'
			// empty
			case ' ', '\t', '\r', '\n':
				return '_'
			}
			if validator.IsPrintable(string(r)) {
				return r
			}
			return '_'
		}, text), 0, 64)
		text = strings.Join(strings.FieldsFunc(text, func(r rune) bool {
			return r == '_' || r == ' '
		}), "_")
		return text
	}()

	if filename == "" {
		filename = fmt.Sprintf("%d_%s", message.GetID(), xid.New().String())
	}
	return filename + ext
}

func BuildCancelButton(taskID string) tg.KeyboardButtonClass {
	return &tg.KeyboardButtonCallback{
		Text: "取消任务",
		Data: fmt.Appendf(nil, "cancel %s", taskID),
	}
}

func InputMessageClassSliceFromInt(ids []int) []tg.InputMessageClass {
	result := make([]tg.InputMessageClass, 0, len(ids))
	for _, id := range ids {
		result = append(result, &tg.InputMessageID{
			ID: id,
		})
	}
	return result
}

func GetMessagesRange(ctx *ext.Context, chatID int64, minId, maxId int) ([]*tg.Message, error) {
	if minId > maxId {
		return nil, fmt.Errorf("minId (%d) cannot be greater than maxId (%d)", minId, maxId)
	}
	total := maxId - minId + 1
	msgIds := mathutil.Range(minId, total)
	toFetchIds := make([]int, 0, total)
	cached := make(map[int]*tg.Message, total)
	for _, id := range msgIds {
		if msg, ok := cache.Get[*tg.Message](fmt.Sprintf("tgmsg:%d:%d:%d", ctx.Self.ID, chatID, id)); ok {
			cached[id] = msg
		} else {
			toFetchIds = append(toFetchIds, id)
		}
	}
	if len(toFetchIds) == 0 {
		return maputil.Values(cached), nil
	}

	result := make([]*tg.Message, 0, total)
	chunks := slice.Chunk(toFetchIds, 100)
	for _, chunk := range chunks {
		msgs, err := ctx.GetMessages(chatID, InputMessageClassSliceFromInt(chunk))
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

	for _, msg := range result {
		cache.Set(fmt.Sprintf("tgmsg:%d:%d:%d", ctx.Self.ID, chatID, msg.GetID()), msg)
	}
	for _, msg := range cached {
		if msg == nil {
			continue
		}
		result = append(result, msg)
	}
	return result, nil
}

func GetMessageByID(ctx *ext.Context, chatID int64, msgID int) (*tg.Message, error) {
	key := fmt.Sprintf("tgmsg:%d:%d:%d", ctx.Self.ID, chatID, msgID)
	if msg, ok := cache.Get[*tg.Message](key); ok {
		return msg, nil
	}
	msgs, err := ctx.GetMessages(chatID, []tg.InputMessageClass{
		&tg.InputMessageID{ID: msgID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get message by ID: %w", err)
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message not found: chatID=%d, msgID=%d", chatID, msgID)
	}
	msg := msgs[0]
	tgm, ok := msg.(*tg.Message)
	if !ok {
		return nil, fmt.Errorf("unexpected message type: %T", msg)
	}
	cache.Set(key, tgm)
	return tgm, nil
}
