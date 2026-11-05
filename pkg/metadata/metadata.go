package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gotd/td/constant"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

// Version is the current normalized message metadata schema.
const Version = 1

// MetaSuffix is appended to a saved file when sidecar export is enabled.
const MetaSuffix = ".meta.json"

type ChatInfo struct {
	ID       int64  `json:"id"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

type SenderInfo struct {
	ID       int64  `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
}

type ReplyInfo struct {
	ChatID    int64 `json:"chat_id,omitempty"`
	MessageID int   `json:"message_id,omitempty"`
	TopicID   int   `json:"topic_id,omitempty"`
}

type ForwardInfo struct {
	Date         string `json:"date,omitempty"`
	ChatID       int64  `json:"chat_id,omitempty"`
	ChatTitle    string `json:"chat_title,omitempty"`
	ChatUsername string `json:"chat_username,omitempty"`
	MessageID    int    `json:"message_id,omitempty"`
	Author       string `json:"author,omitempty"`
}

type MessageInfo struct {
	ID          int          `json:"id"`
	Date        string       `json:"date,omitempty"`
	EditDate    string       `json:"edit_date,omitempty"`
	Chat        ChatInfo     `json:"chat"`
	TopicID     int          `json:"topic_id,omitempty"`
	Sender      *SenderInfo  `json:"sender,omitempty"`
	Text        string       `json:"text,omitempty"`
	ReplyTo     *ReplyInfo   `json:"reply_to,omitempty"`
	ForwardFrom *ForwardInfo `json:"forward_from,omitempty"`
	GroupedID   int64        `json:"grouped_id,omitempty"`
	Link        string       `json:"link,omitempty"`
}

type FileInfo struct {
	Name         string  `json:"name,omitempty"`
	OriginalName string  `json:"original_name,omitempty"`
	MediaType    string  `json:"media_type,omitempty"`
	MimeType     string  `json:"mime_type,omitempty"`
	Size         int64   `json:"size,omitempty"`
	Width        int     `json:"width,omitempty"`
	Height       int     `json:"height,omitempty"`
	Duration     float64 `json:"duration,omitempty"`
	Title        string  `json:"title,omitempty"`
	Performer    string  `json:"performer,omitempty"`
	StickerEmoji string  `json:"sticker_emoji,omitempty"`
}

type FileMetadata struct {
	Version int         `json:"version"`
	Message MessageInfo `json:"message"`
	File    FileInfo    `json:"file"`
	Tags    []string    `json:"tags,omitempty"`
}

// BuildFromMessage normalizes the Telegram message fields stored for a saved
// file. It never stores access hashes or raw Telegram objects.
func BuildFromMessage(ctx context.Context, msg *tg.Message, savedName string, fileSize int64, includeText bool) FileMetadata {
	meta := FileMetadata{
		Version: Version,
		File:    FileInfo{Name: savedName, Size: fileSize},
	}
	if msg == nil {
		return meta
	}

	meta.Message.ID = msg.GetID()
	meta.Message.Date = unixTime(msg.GetDate())
	if editDate, ok := msg.GetEditDate(); ok {
		meta.Message.EditDate = unixTime(editDate)
	}
	if groupedID, ok := msg.GetGroupedID(); ok {
		meta.Message.GroupedID = groupedID
	}

	peer := msg.GetPeerID()
	chatID := tgutil.ChatIdFromPeer(peer)
	chat := tgutil.ChatInfoFromExt(tgutil.ExtFromContext(ctx), peer)
	meta.Message.Chat = ChatInfo{ID: chatID, Title: chat.Title, Username: chat.Username}
	meta.Message.Link = messageLink(chatID, chat.Username, msg.GetID())

	if includeText {
		meta.Message.Text = msg.GetMessage()
		meta.Tags = strutil.ExtractTagsFromText(meta.Message.Text)
	}

	if from, ok := msg.GetFromID(); ok {
		info := tgutil.ChatInfoFromExt(tgutil.ExtFromContext(ctx), from)
		name := info.Title
		if name == "" {
			name, _ = msg.GetPostAuthor()
		}
		meta.Message.Sender = &SenderInfo{
			ID:       tgutil.ChatIdFromPeer(from),
			Name:     name,
			Username: info.Username,
		}
	} else if author, ok := msg.GetPostAuthor(); ok && author != "" {
		meta.Message.Sender = &SenderInfo{Name: author}
	}

	if reply, ok := msg.GetReplyTo(); ok {
		if header, ok := reply.(*tg.MessageReplyHeader); ok {
			info := &ReplyInfo{MessageID: header.ReplyToMsgID}
			if peerID, ok := header.GetReplyToPeerID(); ok {
				info.ChatID = tgutil.ChatIdFromPeer(peerID)
			}
			if topicID, ok := header.GetReplyToTopID(); ok {
				info.TopicID = topicID
				meta.Message.TopicID = topicID
			} else if header.ForumTopic {
				meta.Message.TopicID = header.ReplyToMsgID
			}
			meta.Message.ReplyTo = info
		}
	}

	if fwd, ok := msg.GetFwdFrom(); ok {
		info := &ForwardInfo{
			Date:      unixTime(fwd.Date),
			MessageID: fwd.ChannelPost,
			Author:    fwd.PostAuthor,
			ChatID:    tgutil.ChatIdFromPeer(fwd.FromID),
		}
		if fwd.FromID != nil {
			chat := tgutil.ChatInfoFromExt(tgutil.ExtFromContext(ctx), fwd.FromID)
			info.ChatTitle, info.ChatUsername = chat.Title, chat.Username
		}
		if info.Author == "" {
			info.Author = fwd.FromName
		}
		if savedPeer, ok := fwd.GetSavedFromPeer(); ok {
			info.ChatID = tgutil.ChatIdFromPeer(savedPeer)
			chat := tgutil.ChatInfoFromExt(tgutil.ExtFromContext(ctx), savedPeer)
			info.ChatTitle, info.ChatUsername = chat.Title, chat.Username
		}
		if fwd.SavedFromMsgID != 0 {
			info.MessageID = fwd.SavedFromMsgID
		}
		meta.Message.ForwardFrom = info
	}

	meta.File.MediaType, meta.File.OriginalName = mediaInfo(msg.Media)
	meta.File.MimeType, meta.File.Width, meta.File.Height, meta.File.Duration,
		meta.File.Title, meta.File.Performer, meta.File.StickerEmoji = documentInfo(msg.Media)
	return meta
}

func unixTime(seconds int) string {
	if seconds == 0 {
		return ""
	}
	return time.Unix(int64(seconds), 0).UTC().Format(time.RFC3339)
}

func messageLink(chatID int64, username string, messageID int) string {
	if messageID == 0 {
		return ""
	}
	if username != "" {
		return fmt.Sprintf("https://t.me/%s/%d", username, messageID)
	}
	peerID := constant.TDLibPeerID(chatID)
	if peerID.IsChannel() {
		return fmt.Sprintf("https://t.me/c/%d/%d", peerID.ToPlain(), messageID)
	}
	return ""
}

func mediaInfo(media tg.MessageMediaClass) (mediaType, originalName string) {
	if media == nil {
		return "", ""
	}
	originalName, _ = tgutil.GetMediaFileName(media)
	switch media.(type) {
	case *tg.MessageMediaDocument:
		return "document", originalName
	case *tg.MessageMediaPhoto:
		return "photo", originalName
	default:
		return "", originalName
	}
}

func documentInfo(media tg.MessageMediaClass) (mimeType string, width, height int, duration float64, title, performer, stickerEmoji string) {
	mediaDoc, ok := media.(*tg.MessageMediaDocument)
	if !ok {
		if photoMedia, ok := media.(*tg.MessageMediaPhoto); ok {
			if photo, ok := photoMedia.Photo.AsNotEmpty(); ok {
				for _, size := range photo.Sizes {
					switch s := size.(type) {
					case *tg.PhotoSize:
						if s.W > width || (s.W == width && s.H > height) {
							width, height = s.W, s.H
						}
					case *tg.PhotoSizeProgressive:
						if s.W > width || (s.W == width && s.H > height) {
							width, height = s.W, s.H
						}
					}
				}
			}
		}
		return
	}
	doc, ok := mediaDoc.Document.AsNotEmpty()
	if !ok {
		return
	}
	mimeType = doc.MimeType
	for _, attr := range doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeVideo:
			duration = a.Duration
			width, height = a.W, a.H
		case *tg.DocumentAttributeAudio:
			duration = float64(a.Duration)
			title, performer = a.Title, a.Performer
		case *tg.DocumentAttributeImageSize:
			if width == 0 {
				width, height = a.W, a.H
			}
		case *tg.DocumentAttributeSticker:
			stickerEmoji = a.Alt
		}
	}
	return
}

func (m FileMetadata) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// BuildJSON builds and serializes metadata for one saved file.
func BuildJSON(ctx context.Context, msg *tg.Message, savedName string, fileSize int64, includeText bool) (string, int, error) {
	meta := BuildFromMessage(ctx, msg, savedName, fileSize, includeText)
	raw, err := meta.ToJSON()
	if err != nil {
		return "", 0, err
	}
	return string(raw), Version, nil
}
