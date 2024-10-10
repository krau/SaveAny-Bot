package bot

import (
	"fmt"
	"regexp"

	"github.com/krau/SaveAny-Bot/config"
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

func AddTaskReplyMarkup(messageID int) *telego.InlineKeyboardMarkup {
	storageButtons := make([]telego.InlineKeyboardButton, 0)
	if config.Cfg.Storage.Local.Enable {
		storageButtons = append(storageButtons, telegoutil.InlineKeyboardButton("服务器磁盘").
			WithCallbackData(fmt.Sprintf("add %d local", messageID)))
	}
	if config.Cfg.Storage.Alist.Enable {
		storageButtons = append(storageButtons, telegoutil.InlineKeyboardButton("Alist").
			WithCallbackData(fmt.Sprintf("add %d alist", messageID)))
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
