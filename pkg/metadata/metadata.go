package metadata

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

type ForwardInfo struct {
	Date         string `json:"date,omitempty"`
	ChatID       int64  `json:"chat_id,omitempty"`
	ChatTitle    string `json:"chat_title,omitempty"`
	ChatUsername string `json:"chat_username,omitempty"`
	MessageID    int    `json:"message_id,omitempty"`
	Author       string `json:"author,omitempty"`
}

type ReplyInfo struct {
	MsgID  int   `json:"msg_id,omitempty"`
	ChatID int64 `json:"chat_id,omitempty"`
}

type FileMetadata struct {
	MessageID    int          `json:"message_id"`
	Date         string       `json:"date"`
	EditDate     string       `json:"edit_date,omitempty"`
	ChatID       int64        `json:"chat_id"`
	ChatTitle    string       `json:"chat_title,omitempty"`
	ChatUsername string       `json:"chat_username,omitempty"`
	SenderID     int64        `json:"sender_id,omitempty"`
	Text         string       `json:"text,omitempty"`
	MediaType    string       `json:"media_type"`
	FileName     string       `json:"file_name,omitempty"`
	FileSize     int64        `json:"file_size,omitempty"`
	MimeType     string       `json:"mime_type,omitempty"`
	Width        int          `json:"width,omitempty"`
	Height       int          `json:"height,omitempty"`
	Duration     float64      `json:"duration,omitempty"`
	Title        string       `json:"title,omitempty"`
	Performer    string       `json:"performer,omitempty"`
	ForwardFrom  *ForwardInfo `json:"forward_from,omitempty"`
	ReplyTo      *ReplyInfo   `json:"reply_to,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	GroupID      int64        `json:"group_id,omitempty"`
	OriginalName string       `json:"original_name,omitempty"`
}

func BuildFromMessage(ctx context.Context, msg *tg.Message, fileName string, fileSize int64) FileMetadata {
	m := FileMetadata{
		MessageID: msg.GetID(),
		Date: func() string {
			d := msg.GetDate()
			if d == 0 {
				return ""
			}
			return time.Unix(int64(d), 0).UTC().Format(time.RFC3339)
		}(),
		ChatID:   tgutil.ChatIdFromPeer(msg.GetPeerID()),
		Text:     msg.GetMessage(),
		FileSize: fileSize,
		GroupID:  func() int64 { id, _ := msg.GetGroupedID(); return id }(),
	}

	m.ChatTitle, m.ChatUsername = tgutil.ChatInfoFromExt(tgutil.ExtFromContext(ctx), msg.GetPeerID())

	// media type, mime type, original name, file attributes
	if msg.Media != nil {
		switch media := msg.Media.(type) {
		case *tg.MessageMediaDocument:
			m.MediaType = "document"
			if doc, ok := media.Document.AsNotEmpty(); ok {
				m.MimeType = doc.MimeType
				for _, attr := range doc.Attributes {
					switch a := attr.(type) {
					case *tg.DocumentAttributeVideo:
						m.Duration = a.GetDuration()
						m.Width = a.GetW()
						m.Height = a.GetH()
					case *tg.DocumentAttributeAudio:
						m.Duration = float64(a.GetDuration())
						if title, ok := a.GetTitle(); ok {
							m.Title = title
						}
						if performer, ok := a.GetPerformer(); ok {
							m.Performer = performer
						}
					case *tg.DocumentAttributeImageSize:
						if m.Width == 0 {
							m.Width = a.GetW()
							m.Height = a.GetH()
						}
					}
				}
			}
		case *tg.MessageMediaPhoto:
			m.MediaType = "photo"
			if photo, ok := media.Photo.AsNotEmpty(); ok {
				for _, size := range photo.Sizes {
					switch s := size.(type) {
					case *tg.PhotoSize:
						if s.W > m.Width || (s.W == m.Width && s.H > m.Height) {
							m.Width, m.Height = s.W, s.H
						}
					case *tg.PhotoSizeProgressive:
						if s.W > m.Width || (s.W == m.Width && s.H > m.Height) {
							m.Width, m.Height = s.W, s.H
						}
					}
				}
			}
		}
		origName, _ := tgutil.GetMediaFileName(msg.Media)
		m.OriginalName = origName
	}

	// file name from the tfile layer (after applying user strategy)
	m.FileName = fileName

	// tags from message text
	if tags := strutil.ExtractTagsFromText(msg.GetMessage()); len(tags) > 0 {
		m.Tags = tags
	}

	// sender id
	if from, ok := msg.GetFromID(); ok {
		m.SenderID = tgutil.ChatIdFromPeer(from)
	}

	// edit date
	if d, ok := msg.GetEditDate(); ok && d != 0 {
		m.EditDate = time.Unix(int64(d), 0).UTC().Format(time.RFC3339)
	}

	// reply info
	if reply, ok := msg.GetReplyTo(); ok {
		if header, ok := reply.(*tg.MessageReplyHeader); ok {
			msgID, _ := header.GetReplyToMsgID()
			m.ReplyTo = &ReplyInfo{
				MsgID: msgID,
			}
			if peerID, ok := header.GetReplyToPeerID(); ok {
				m.ReplyTo.ChatID = tgutil.ChatIdFromPeer(peerID)
			}
		}
	}

	// forward info
	if fwd, ok := msg.GetFwdFrom(); ok {
		fwdDate := fwd.GetDate()
		fi := &ForwardInfo{
			Date: func() string {
				if fwdDate == 0 {
					return ""
				}
				return time.Unix(int64(fwdDate), 0).UTC().Format(time.RFC3339)
			}(),
		}
		if fromID, ok := fwd.GetFromID(); ok {
			fi.ChatID = tgutil.ChatIdFromPeer(fromID)
			fi.ChatTitle, fi.ChatUsername = tgutil.ChatInfoFromExt(tgutil.ExtFromContext(ctx), fromID)
		}
		if author, ok := fwd.GetPostAuthor(); ok {
			fi.Author = author
		}
		if postID, ok := fwd.GetChannelPost(); ok {
			fi.MessageID = postID
		}
		m.ForwardFrom = fi
	}

	return m
}

func (m FileMetadata) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// MetaSuffix is the file extension appended to metadata sidecar files.
const MetaSuffix = ".meta.json"
