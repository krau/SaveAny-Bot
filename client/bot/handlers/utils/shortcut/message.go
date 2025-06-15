// Some shortcuts for duplicate code in handlers, they should return dispatcher errors
package shortcut

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/types"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/common/utils/tphutil"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

// 获取消息中的文件并回复等待消息, 返回等待消息, 获取到的文件
func GetFileFromMessageWithReply(ctx *ext.Context, update *ext.Update, message *tg.Message, tfileopts ...tfile.TGFileOptions) (replied *types.Message,
	file tfile.TGFileMessage, err error,
) {
	logger := log.FromContext(ctx)
	media := message.Media
	supported := mediautil.IsSupported(media)
	if !supported {
		ctx.Reply(update, ext.ReplyTextString("不支持的消息类型"), nil)
		return nil, nil, dispatcher.EndGroups
	}

	replied, err = ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		logger.Errorf("Failed to reply: %s", err)
		return nil, nil, dispatcher.EndGroups
	}
	options := []tfile.TGFileOptions{
		tfile.WithMessage(message),
	}
	if len(tfileopts) > 0 {
		options = append(options, tfileopts...)
	} else {
		options = append(options, tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*message)))
	}
	file, err = tfile.FromMediaMessage(media, message, options...)
	if err != nil {
		logger.Errorf("Failed to get file from media: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取文件失败: "+err.Error()), nil)
		return nil, nil, dispatcher.EndGroups
	}
	return replied, file, nil
}

type EditMessageFunc func(text string, markup tg.ReplyMarkupClass)

// 获取链接中的文件并回复等待消息
func GetFilesFromUpdateLinkMessageWithReplyEdit(ctx *ext.Context, update *ext.Update) (replied *types.Message, files []tfile.TGFileMessage, editReplied EditMessageFunc, err error) {
	logger := log.FromContext(ctx)
	msgLinks := re.TgMessageLinkRegexp.FindAllString(update.EffectiveMessage.GetMessage(), -1)
	if len(msgLinks) == 0 {
		logger.Warn("no matched message links but called handleMessageLink")
		return nil, nil, nil, dispatcher.EndGroups
	}
	replied, err = ctx.Reply(update, ext.ReplyTextString("正在获取消息..."), nil)
	if err != nil {
		logger.Errorf("failed to reply: %s", err)
		return nil, nil, nil, dispatcher.EndGroups
	}
	editReplied = func(text string, markup tg.ReplyMarkupClass) {
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:          replied.ID,
			Message:     text,
			ReplyMarkup: markup,
		}); err != nil {
			logger.Errorf("failed to edit message: %s", err)
		}
	}

	files = make([]tfile.TGFileMessage, 0, len(msgLinks))
	for _, link := range msgLinks {
		chatId, msgId, err := tgutil.ParseMessageLink(ctx, link)
		if err != nil {
			logger.Errorf("failed to parse message link %s: %s", link, err)
			continue
		}
		msg, err := tgutil.GetMessageByID(ctx, chatId, msgId)
		if err != nil {
			logger.Errorf("failed to get message by ID: %s", err)
			continue
		}
		media, ok := msg.GetMedia()
		if !ok {
			logger.Debugf("message %d has no media", msg.GetID())
			continue
		}
		file, err := tfile.FromMediaMessage(media, msg, tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*msg)))
		if err != nil {
			logger.Errorf("failed to create file from media: %s", err)
			continue
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		editReplied("没有找到可保存的文件", nil)
		return nil, nil, nil, dispatcher.EndGroups
	}
	return replied, files, editReplied, nil
}

func GetCallbackDataWithAnswer[DataType any](ctx *ext.Context, update *ext.Update, dataid string) (DataType, error) {
	data, ok := cache.Get[DataType](dataid)
	if !ok {
		log.FromContext(ctx).Warnf("Invalid data ID: %s", dataid)
		queryID := update.CallbackQuery.GetQueryID()
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "数据已过期或无效"))
		var zero DataType
		return zero, dispatcher.EndGroups
	}
	return data, nil
}

type TelegraphResult struct {
	Pics   []string        `json:"pics"`    // image urls
	TphDir string          `json:"tph_dir"` // telegraph path, unescaped
	Page   *telegraph.Page `json:"page"`    // telegraph page node
}

// return replied message, image urls, telegraph path(unescaped), error
func GetTphPicsFromMessageWithReply(ctx *ext.Context, update *ext.Update) (*types.Message, *TelegraphResult, error) {
	logger := log.FromContext(ctx)
	tphurl := re.TelegraphUrlRegexp.FindString(update.EffectiveMessage.GetMessage()) // TODO: batch urls
	if tphurl == "" {
		logger.Warnf("No telegraph url found but called handleTelegraph")
		return nil, nil, dispatcher.ContinueGroups
	}
	pagepath := strings.Split(tphurl, "/")[len(strings.Split(tphurl, "/"))-1]
	tphdir, err := url.PathUnescape(pagepath)
	if err != nil {
		logger.Errorf("Failed to unescape telegraph path: %s", err)
		ctx.Reply(update, ext.ReplyTextString("解析 telegraph 路径失败: "+err.Error()), nil)
		return nil, nil, dispatcher.EndGroups
	}
	msg, err := ctx.Reply(update, ext.ReplyTextString("正在获取 telegraph 页面..."), nil)
	if err != nil {
		logger.Errorf("Failed to reply to update: %s", err)
		return nil, nil, dispatcher.EndGroups
	}
	page, err := tphutil.DefaultClient().GetPage(ctx, pagepath)
	if err != nil {
		logger.Errorf("Failed to get telegraph page: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取 telegraph 页面失败: "+err.Error()), nil)
		return nil, nil, dispatcher.EndGroups
	}
	imgs := make([]string, 0)
	for _, elem := range page.Content {
		var node telegraph.NodeElement
		data, err := json.Marshal(elem)
		if err != nil {
			logger.Errorf("Failed to marshal element: %s", err)
			continue
		}
		err = json.Unmarshal(data, &node)
		if err != nil {
			logger.Errorf("Failed to unmarshal element: %s", err)
			continue
		}

		if len(node.Children) != 0 {
			for _, child := range node.Children {
				imgs = append(imgs, tphutil.GetNodeImages(child)...)
			}
		}
		if node.Tag == "img" {
			if src, ok := node.Attrs["src"]; ok {
				imgs = append(imgs, src)
			}
		}
	}
	if len(imgs) == 0 {
		logger.Warn("No images found in telegraph page")
		ctx.Reply(update, ext.ReplyTextString("在 telegraph 页面中未找到图片"), nil)
		return nil, nil, dispatcher.EndGroups
	}
	return msg, &TelegraphResult{
		Pics:   imgs,
		TphDir: tphdir,
		Page:   page,
	}, nil
}
