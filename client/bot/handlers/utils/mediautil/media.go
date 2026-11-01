package mediautil

import (
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/fnamest"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

func IsSupported(media tg.MessageMediaClass) bool {
	switch media.(type) {
	case *tg.MessageMediaDocument, *tg.MessageMediaPhoto:
		return true
	default:
		return false
	}
}

type FilenameTemplateData struct {
	MsgID    string `json:"msgid,omitempty"`
	MsgTags  string `json:"msgtags,omitempty"`
	MsgGen   string `json:"msggen,omitempty"`
	MsgDate  string `json:"msgdate,omitempty"`
	MsgRaw   string `json:"msgraw,omitempty"`
	OrigName string `json:"origname,omitempty"`
	ChatID   string `json:"chatid,omitempty"`
}

func (f FilenameTemplateData) ToMap() map[string]string {
	return map[string]string{
		"msgid":    f.MsgID,
		"msgtags":  f.MsgTags,
		"msggen":   f.MsgGen,
		"msgraw":   f.MsgRaw,
		"msgdate":  f.MsgDate,
		"origname": f.OrigName,
		"chatid":   f.ChatID,
	}
}

func TfileOptions(ctx context.Context, user *database.User, message *tg.Message) []tfile.TGFileOption {
	opts := make([]tfile.TGFileOption, 0)
	opts = append(opts, tfile.WithName(FilenameFor(ctx, user, message)), tfile.WithMessage(message))
	return opts
}

// FilenameFor resolves the configured filename strategy for one message.
func FilenameFor(ctx context.Context, user *database.User, message *tg.Message) string {
	switch user.FilenameStrategy {
	case fnamest.Message.String():
		return tgutil.GenFileNameFromMessage(*message)
	case fnamest.Template.String():
		if user.FilenameTemplate == "" {
			log.FromContext(ctx).Warnf("empty filename template")
			return tgutil.GenFileNameFromMessage(*message)
		}
		tmpl, err := template.New("filename").Parse(user.FilenameTemplate)
		if err != nil {
			log.FromContext(ctx).Errorf("failed to parse filename template: %s", err)
			return tgutil.GenFileNameFromMessage(*message)
		}
		data := BuildFilenameTemplateData(message)
		var sb strings.Builder
		if err := tmpl.Execute(&sb, data); err != nil {
			log.FromContext(ctx).Errorf("failed to execute filename template: %s", err)
			return tgutil.GenFileNameFromMessage(*message)
		}
		if sb.String() == "" {
			return tgutil.GenFileNameFromMessage(*message)
		}
		return sb.String()
	default:
		return tgutil.GenFileNameFromMessage(*message)
	}
}

// ApplyAlbumCaptions shares one album caption with the items that carry none,
// so filename templates and message rules see the same text for every file in
// the group.
func ApplyAlbumCaptions(ctx context.Context, user *database.User, files []tfile.TGFileMessage) []tfile.TGFileMessage {
	captions := make(map[int64]string)
	for _, file := range files {
		message := file.Message()
		if message == nil {
			continue
		}
		groupID, grouped := message.GetGroupedID()
		if !grouped || groupID == 0 {
			continue
		}
		if captions[groupID] == "" && strings.TrimSpace(message.GetMessage()) != "" {
			captions[groupID] = message.GetMessage()
		}
	}
	if len(captions) == 0 {
		return files
	}
	// Track names already used in each group so a template without a unique
	// field cannot collapse several album items onto the same path.
	usedNames := make(map[int64]map[string]struct{})
	for _, file := range files {
		message := file.Message()
		if message == nil {
			continue
		}
		groupID, grouped := message.GetGroupedID()
		if !grouped || groupID == 0 {
			continue
		}
		if usedNames[groupID] == nil {
			usedNames[groupID] = make(map[string]struct{})
		}
		usedNames[groupID][file.Name()] = struct{}{}
	}

	result := make([]tfile.TGFileMessage, 0, len(files))
	for _, file := range files {
		message := file.Message()
		if message != nil {
			if groupID, grouped := message.GetGroupedID(); grouped && groupID != 0 {
				if caption := captions[groupID]; caption != "" && strings.TrimSpace(message.GetMessage()) == "" {
					file = applyAlbumCaption(ctx, user, file, caption, usedNames[groupID])
				}
			}
		}
		result = append(result, file)
	}
	return result
}

type captionOverride struct {
	tfile.TGFileMessage
	message *tg.Message
}

func (f *captionOverride) Message() *tg.Message {
	return f.message
}

func applyAlbumCaption(ctx context.Context, user *database.User, file tfile.TGFileMessage, caption string, used map[string]struct{}) tfile.TGFileMessage {
	message := file.Message()
	copied := *message
	copied.Message = caption
	if name := FilenameFor(ctx, user, &copied); name != "" {
		if _, exists := used[name]; !exists {
			used[name] = struct{}{}
			file.SetName(name)
		}
	}
	return &captionOverride{TGFileMessage: file, message: &copied}
}

func BuildFilenameTemplateData(message *tg.Message) map[string]string {
	data := FilenameTemplateData{
		MsgID: func() string {
			id := message.GetID()
			if id == 0 {
				return ""
			}
			return fmt.Sprintf("%d", id)
		}(),
		MsgTags: func() string {
			tags := strutil.ExtractTagsFromText(message.GetMessage())
			if len(tags) == 0 {
				return ""
			}
			return strings.Join(tags, "_")
		}(),
		MsgGen: tgutil.GenFileNameFromMessage(*message),
		OrigName: func() string {
			f, _ := tgutil.GetMediaFileName(message.Media)
			return f
		}(),
		MsgDate: func() string {
			date := message.GetDate()
			if date == 0 {
				return ""
			}
			t := time.Unix(int64(date), 0)
			return t.Format("2006-01-02_15-04-05")
		}(),
		MsgRaw: message.GetMessage(),
		ChatID: func() string {
			// 如果消息是频道的(从消息链接中fetch的) 直接使用其chat id,
			// 无论它是否是从其他来源转发的
			if message.GetPost() {
				peer := message.GetPeerID()
				switch p := peer.(type) {
				case *tg.PeerChannel:
					return intToStringOmitZero(p.ChannelID)
				default: // impossible case
					return intToStringOmitZero(tgutil.ChatIdFromPeer(peer))
				}
			}
			fwdHeader, ok := message.GetFwdFrom()
			if !ok {
				return intToStringOmitZero(tgutil.ChatIdFromPeer(message.GetPeerID()))
			}
			fwdFrom, ok := fwdHeader.GetFromID()
			if !ok {
				return intToStringOmitZero(tgutil.ChatIdFromPeer(message.GetPeerID()))
			}
			return intToStringOmitZero(tgutil.ChatIdFromPeer(fwdFrom))
		}(),
	}.ToMap()
	return data
}

func intToStringOmitZero(i int64) string {
	if i == 0 {
		return ""
	}
	return fmt.Sprintf("%d", i)
}
