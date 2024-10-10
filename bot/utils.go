package bot

import (
	"fmt"
	"regexp"

	"github.com/krau/SaveAny-Bot/storage"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

func FileDownloadURL(filePath string) string {
	return Bot.FileDownloadURL(filePath)
}

func ReplyMessage(replyTo telego.Message, format string, args ...any) (*telego.Message, error) {
	return Bot.SendMessage(telegoutil.Messagef(replyTo.Chat.ChatID(), format, args...).
		WithReplyParameters(&telego.ReplyParameters{
			MessageID: replyTo.MessageID,
		}))
}

var StorageDisplayNames = map[string]string{
	"all":    "全部",
	"local":  "服务器磁盘",
	"alist":  "Alist",
	"webdav": "WebDAV",
}

func AddTaskReplyMarkup(messageID int) *telego.InlineKeyboardMarkup {
	storageButtons := make([]telego.InlineKeyboardButton, 0)
	for name := range storage.Storages {
		storageButtons = append(storageButtons, telegoutil.InlineKeyboardButton(StorageDisplayNames[string(name)]).
			WithCallbackData(fmt.Sprintf("add %d %s", messageID, name)))
	}

	if len(storageButtons) > 1 {
		return telegoutil.InlineKeyboard(storageButtons, []telego.InlineKeyboardButton{
			telegoutil.InlineKeyboardButton("全部").WithCallbackData(fmt.Sprintf("add %d all", messageID)),
		})
	}
	if len(storageButtons) == 1 {
		return telegoutil.InlineKeyboard(storageButtons)
	}
	return nil
}

var markdownRe = regexp.MustCompile("([" + regexp.QuoteMeta(`\_*[]()~`+"`"+`>#+-=|{}.!`) + "])")

func EscapeMarkdown(text string) string {
	return markdownRe.ReplaceAllString(text, "\\$1")
}
