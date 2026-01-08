package tgutil

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/mathutil"
	"github.com/duke-git/lancet/v2/slice"
	lcstrutil "github.com/duke-git/lancet/v2/strutil"
	"github.com/duke-git/lancet/v2/validator"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gotd/td/constant"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/rs/xid"
)

// generate a file name from the message content and media type
//
// it will never return an empty string
func GenFileNameFromMessage(message tg.Message) string {
	ext := func(media tg.MessageMediaClass) string {
		switch media := media.(type) {
		case *tg.MessageMediaDocument:
			doc, ok := media.Document.AsNotEmpty()
			if !ok {
				return ""
			}
			mmt := mimetype.Lookup(doc.MimeType)
			if mmt == nil || mmt.Extension() == "" {
				return ""
			}
			return mmt.Extension()
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
			switch r {
			case '/', '\\',
				':', '*', '?', '"', '<', '>', '|':
				return '_'
			}
			if unicode.IsControl(r) || unicode.IsSpace(r) {
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
		mname, err := GetMediaFileName(message.Media)
		if err != nil {
			filename = fmt.Sprintf("%d_%s", message.GetID(), xid.New().String())
		} else {
			filename = mname
		}

	}
	return filename + ext
}

func BuildCancelButton(taskID string) tg.KeyboardButtonClass {
	return &tg.KeyboardButtonCallback{
		Text: i18n.T(i18nk.BotMsgCommonCancelButtonText, nil),
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

func GetMessagesRange(ctx *ext.Context, chatID int64, minId, maxId int) (msg []*tg.Message, err error) {
	if msg, err = getMessagesRange(ctx, chatID, minId, maxId); err == nil {
		return
	}
	in := constant.TDLibPeerID(chatID)
	plain := in.ToPlain()
	var channel constant.TDLibPeerID
	channel.Channel(plain)
	if msg, err = getMessagesRange(ctx, int64(channel), minId, maxId); err == nil {
		return
	}
	var userID constant.TDLibPeerID
	userID.User(plain)
	if msg, err = getMessagesRange(ctx, int64(userID), minId, maxId); err == nil {
		return
	}
	var chat constant.TDLibPeerID
	chat.Chat(plain)
	if msg, err = getMessagesRange(ctx, int64(chat), minId, maxId); err == nil {
		return
	}
	return nil, fmt.Errorf("failed to get messages range for chat %d: %w", chatID, err)
}

func getMessagesRange(ctx *ext.Context, chatID int64, minId, maxId int) ([]*tg.Message, error) {
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

// [TODO]
// type MessageItem struct {
// 	Message *tg.Message
// 	Error   error
// }

// func IterMessages(ctx *ext.Context, chatID int64, minId, maxId int) (<-chan MessageItem, error) {
// 	total := maxId - minId + 1
// 	ch := make(chan MessageItem, 100)

// 	go func() {
// 		defer close(ch)
// 		if !ctx.Self.Bot {
// 			perr := ctx.PeerStorage.GetInputPeerById(chatID)
// 			if perr == nil || perr.(*tg.InputPeerEmpty) != nil {
// 				ch <- MessageItem{
// 					Error: fmt.Errorf("peer not found: %d", chatID),
// 				}
// 				return
// 			}

// 			for i := 0; i < total; i += 100 {
// 				start := minId + i
// 				end := min(start+100, maxId)
// 				msgs, err := ctx.Raw.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
// 					Peer:      perr,
// 					OffsetID:  start,
// 					AddOffset: start - end,
// 					Limit:     100,
// 				})
// 				if err != nil {
// 					ch <- MessageItem{
// 						Error: fmt.Errorf("failed to get messages: %w", err),
// 					}
// 					return
// 				}
// 				var msgClass []tg.MessageClass
// 				switch msgsv := msgs.(type) {
// 				case *tg.MessagesMessages:
// 					msgClass = msgsv.GetMessages()
// 				case *tg.MessagesMessagesSlice:
// 					msgClass = msgsv.GetMessages()
// 				case *tg.MessagesChannelMessages:
// 					msgClass = msgsv.GetMessages()
// 				default:
// 					ch <- MessageItem{
// 						Error: fmt.Errorf("unsupported message type: %T", msgsv),
// 					}
// 					continue
// 				}
// 				for _, msg := range msgClass {
// 					msg, ok := msg.AsNotEmpty()
// 					if !ok {
// 						continue
// 					}
// 					switch msg := msg.(type) {
// 					case *tg.Message:
// 						key := fmt.Sprintf("tgmsg:%d:%d:%d", ctx.Self.ID, chatID, msg.GetID())
// 						cache.Set(key, msg)
// 						ch <- MessageItem{
// 							Message: msg,
// 						}
// 					}
// 				}
// 			}
// 		} else {
// 			for i := 0; i < total; i += 100 {
// 				start := minId + i
// 				end := min(start+100, maxId)
// 				msgs, err := GetMessagesRange(ctx, chatID, start, end)
// 				if err != nil {
// 					ch <- MessageItem{
// 						Error: fmt.Errorf("failed to get messages: %w", err),
// 					}
// 					return
// 				}
// 				for _, msg := range msgs {
// 					if msg == nil {
// 						continue
// 					}
// 					ch <- MessageItem{
// 						Message: msg,
// 					}
// 				}
// 			}
// 		}
// 	}()

// 	return ch, nil
// }

func getMessageByID(ctx *ext.Context, chatID int64, msgID int) (*tg.Message, error) {
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

// f**k gotgproto's breaking changes
func GetMessageByID(ctx *ext.Context, chatID int64, msgID int) (*tg.Message, error) {
	// we don't know what the input chatID is bot api style(e.g. channel with -100 prefix) or plain tdlib style(no any prefix and every id is positive)
	if msg, err := getMessageByID(ctx, chatID, msgID); err == nil {
		return msg, nil
	}
	in := constant.TDLibPeerID(chatID)
	plain := in.ToPlain()
	var channel constant.TDLibPeerID
	channel.Channel(plain)
	if msg, err := getMessageByID(ctx, int64(channel), msgID); err == nil {
		return msg, nil
	}
	var chat constant.TDLibPeerID
	chat.Chat(plain)
	if msg, err := getMessageByID(ctx, int64(chat), msgID); err == nil {
		return msg, nil
	}
	var userID constant.TDLibPeerID
	userID.User(plain)
	if msg, err := getMessageByID(ctx, int64(userID), msgID); err == nil {
		return msg, nil
	}

	return nil, fmt.Errorf("failed to get message by ID: chatID=%d, msgID=%d", chatID, msgID)
}

func GetGroupedMessages(ctx *ext.Context, chatID int64, msg *tg.Message) ([]*tg.Message, error) {
	groupID, isGroup := msg.GetGroupedID()
	if !isGroup || groupID == 0 {
		return nil, fmt.Errorf("message %d is not grouped", msg.GetID())
	}
	msgID := msg.GetID()
	minID := msgID - 10
	maxID := msgID + 10
	if minID < 1 {
		minID = 1
	}
	msgs, err := GetMessagesRange(ctx, chatID, minID, maxID)
	if err != nil {
		return nil, err
	}
	groupedMessages := make([]*tg.Message, 0, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		mgid, isGroup := m.GetGroupedID()
		if isGroup && mgid == groupID {
			groupedMessages = append(groupedMessages, m)
		}
	}
	return groupedMessages, nil
}

func ExtractMessageEntityUrls(msg *tg.Message) []string {
	if len(msg.Entities) == 0 {
		return nil
	}
	msgText := msg.GetMessage()
	if msgText == "" {
		return nil
	}

	runes := []rune(msgText)
	utf16Codes := utf16.Encode(runes)

	var urls []string
	for _, entity := range msg.Entities {
		switch ent := entity.(type) {
		case *tg.MessageEntityTextURL:
			urls = append(urls, ent.GetURL())
		case *tg.MessageEntityURL:
			start := ent.Offset
			end := ent.Offset + ent.Length
			if start < 0 || end > len(utf16Codes) {
				continue
			}
			subRunes := utf16.Decode(utf16Codes[start:end])
			urls = append(urls, string(subRunes))
		}
	}
	return urls
}

func ExtractMessageEntityUrlsText(msg *tg.Message) string {
	if msg == nil {
		return ""
	}
	urls := ExtractMessageEntityUrls(msg)
	if len(urls) == 0 {
		return msg.GetMessage()
	}
	var sb strings.Builder
	for _, url := range urls {
		sb.WriteString(url)
		sb.WriteString(" ")
	}
	return sb.String()
}
