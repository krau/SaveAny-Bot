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
	OrigName string `json:"origname,omitempty"`
}

func (f FilenameTemplateData) ToMap() map[string]string {
	return map[string]string{
		"msgid":    f.MsgID,
		"msgtags":  f.MsgTags,
		"msggen":   f.MsgGen,
		"msgdate":  f.MsgDate,
		"origname": f.OrigName,
	}
}

func TfileOptions(ctx context.Context, user *database.User, message *tg.Message) []tfile.TGFileOption {
	opts := make([]tfile.TGFileOption, 0)
	var fnameOpt tfile.TGFileOption
	switch user.FilenameStrategy {
	case fnamest.Message.String():
		fnameOpt = tfile.WithName(tgutil.GenFileNameFromMessage(*message))
	case fnamest.Template.String():
		if user.FilenameTemplate == "" {
			log.FromContext(ctx).Warnf("empty filename template")
			fnameOpt = tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*message))
			break
		}
		tmpl, err := template.New("filename").Parse(user.FilenameTemplate)
		if err != nil {
			log.FromContext(ctx).Errorf("failed to parse filename template: %s", err)
			fnameOpt = tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*message))
			break
		}
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
		}.ToMap()
		var sb strings.Builder
		err = tmpl.Execute(&sb, data)
		if err != nil {
			log.FromContext(ctx).Errorf("failed to execute filename template: %s", err)
			fnameOpt = tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*message))
			break
		}
		fnameOpt = tfile.WithName(sb.String())
	default:
		fnameOpt = tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*message))
	}
	opts = append(opts, fnameOpt, tfile.WithMessage(message))
	return opts
}
