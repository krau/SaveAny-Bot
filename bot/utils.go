package bot

import (
	"fmt"
	"regexp"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/krau/SaveAny-Bot/storage"
)

var StorageDisplayNames = map[string]string{
	"all":    "全部",
	"local":  "服务器磁盘",
	"alist":  "Alist",
	"webdav": "WebDAV",
}

func AddTaskReplyMarkup(messageID int32) telegram.ReplyMarkup {
	// TODO: sort storage buttons
	storageButtons := make([]telegram.KeyboardButton, 0)
	for name := range storage.Storages {
		storageButtons = append(storageButtons, &telegram.KeyboardButtonCallback{
			Text: StorageDisplayNames[string(name)],
			Data: []byte(fmt.Sprintf("add %d %s", messageID, name)),
		})
	}

	if len(storageButtons) > 1 {
		return &telegram.ReplyInlineMarkup{
			Rows: []*telegram.KeyboardButtonRow{
				{
					Buttons: storageButtons,
				},
				{
					Buttons: []telegram.KeyboardButton{
						&telegram.KeyboardButtonCallback{
							Text: "全部",
							Data: []byte(fmt.Sprintf("add %d all", messageID)),
						},
					},
				},
			},
		}
	}

	if len(storageButtons) == 1 {
		return &telegram.ReplyInlineMarkup{
			Rows: []*telegram.KeyboardButtonRow{
				{
					Buttons: storageButtons,
				},
			},
		}
	}
	return nil
}

var markdownRe = regexp.MustCompile("([" + regexp.QuoteMeta(`\_*[]()~`+"`"+`>#+-=|{}.!`) + "])")

func EscapeMarkdown(text string) string {
	return markdownRe.ReplaceAllString(text, "\\$1")
}
