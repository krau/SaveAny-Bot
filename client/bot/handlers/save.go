package handlers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/dirutil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"

	"github.com/krau/SaveAny-Bot/storage"
)

func handleSaveCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) >= 3 {
		return handleBatchSave(ctx, update, args[1:])
	}
	replyTo := update.EffectiveMessage.ReplyToMessage
	if replyTo == nil || replyTo.Message == nil {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgSaveHelpText)), nil)
		return dispatcher.EndGroups
	}
	userDB, err := database.GetUserByChatID(ctx, update.GetUserChat().GetID())
	if err != nil {
		return err
	}
	opts := mediautil.TfileOptions(ctx, userDB, replyTo.Message)
	if len(args) > 1 {
		// custom filename via command arg
		opts = append(opts, tfile.WithName(strings.Join(args[1:], " ")))
	}
	msg, file, err := shortcut.GetFileFromMessageWithReply(ctx, update, replyTo.Message, opts...)
	if err != nil {
		return err
	}
	userId := update.GetUserChat().GetID()
	stors := storage.GetUserStorages(ctx, userId)
	req, err := msgelem.BuildAddOneSelectStorageMessage(ctx, stors, file, msg.ID)
	if err != nil {
		logger.Errorf("构建存储选择消息失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("构建存储选择消息失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	ctx.EditMessage(update.EffectiveChat().GetID(), req)
	return dispatcher.EndGroups
}

func handleSilentSaveReplied(ctx *ext.Context, update *ext.Update) error {
	args := strings.Split(string(update.EffectiveMessage.Text), " ")
	if len(args) >= 3 {
		return handleBatchSave(ctx, update, args[1:])
	}
	stor := storage.FromContext(ctx)
	replyTo := update.EffectiveMessage.ReplyToMessage
	if replyTo == nil || replyTo.Message == nil {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgSaveHelpText)), nil)
		return dispatcher.EndGroups
	}
	userDB, err := database.GetUserByChatID(ctx, update.GetUserChat().GetID())
	if err != nil {
		return err
	}
	opts := mediautil.TfileOptions(ctx, userDB, replyTo.Message)
	if len(args) > 1 {
		// custom filename via command arg
		opts = append(opts, tfile.WithName(strings.Join(args[1:], " ")))
	}
	msg, file, err := shortcut.GetFileFromMessageWithReply(ctx, update, replyTo.Message, opts...)
	if err != nil {
		return err
	}
	return shortcut.CreateAndAddTGFileTaskWithEdit(ctx, update.GetUserChat().GetID(), stor, dirutil.PathFromContext(ctx), file, msg.GetID())
}

func handleBatchSave(ctx *ext.Context, update *ext.Update, args []string) error {
	chatArg := args[0]
	msgIdRangeArg := args[1]
	var filterStr string
	var filter *regexp.Regexp
	if len(args) > 2 {
		filterStr = args[2]
		var err error
		filter, err = regexp.Compile(filterStr)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString("无效的正则表达式: "+err.Error()), nil)
			return dispatcher.EndGroups
		}
	}
	startID, endID, err := strutil.ParseIntStrRange(msgIdRangeArg, "-")
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("无效的消息ID范围: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	chatID, err := tgutil.ParseChatID(ctx, chatArg)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("无效的ID或用户名: "+err.Error()), nil)
		return dispatcher.EndGroups
	}

	replied, err := ctx.Reply(update, ext.ReplyTextString("正在获取消息..."), nil)
	if err != nil {
		log.FromContext(ctx).Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}

	// [TODO]: generator istead of get all messages
	msgs, err := tgutil.GetMessagesRange(ctx, chatID, int(startID), int(endID))
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("获取消息失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	if len(msgs) == 0 {
		ctx.Reply(update, ext.ReplyTextString("没有找到指定范围内的消息"), nil)
		return dispatcher.EndGroups
	}
	files := make([]tfile.TGFileMessage, 0, len(msgs))
	sb := strings.Builder{}
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		media, ok := msg.GetMedia()
		if !ok {
			continue
		}
		supported := mediautil.IsSupported(media)
		if !supported {
			continue
		}
		file, err := tfile.FromMediaMessage(media, ctx.Raw, msg, tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*msg)))
		if err != nil {
			log.FromContext(ctx).Errorf("获取文件失败: %s", err)
			continue
		}
		if filter != nil {
			sb.Reset()
			sb.WriteString(msg.GetMessage())
			sb.WriteString(" ")
			fn, _ := tgutil.GetMediaFileName(media)
			sb.WriteString(fn)
			if !filter.MatchString(sb.String()) {
				continue
			}
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		ctx.Reply(update, ext.ReplyTextString("没有找到指定范围内的可保存消息"), nil)
		return dispatcher.EndGroups
	}
	stor := storage.FromContext(ctx)
	if stor == nil {
		// not in silent mode
		stors := storage.GetUserStorages(ctx, update.GetUserChat().GetID())
		markup, err := msgelem.BuildAddSelectStorageKeyboard(stors, tcbdata.Add{
			Files: files,
		})
		if err != nil {
			log.FromContext(ctx).Errorf("构建存储选择键盘失败: %s", err)
			ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
				ID:      replied.ID,
				Message: "构建存储选择键盘失败: " + err.Error(),
			})
			return dispatcher.EndGroups
		}
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:          replied.ID,
			Message:     fmt.Sprintf("找到 %d 个文件, 请选择存储位置", len(files)),
			ReplyMarkup: markup,
		})
		return dispatcher.EndGroups
	}
	return shortcut.CreateAndAddBatchTGFileTaskWithEdit(ctx, update.GetUserChat().GetID(), stor, "", files, replied.ID)
}
