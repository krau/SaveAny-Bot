package handlers

import (
	"fmt"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tftask"
	"github.com/krau/SaveAny-Bot/pkg/tfile"

	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func handleSaveCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(string(update.EffectiveMessage.Text), " ")
	if len(args) >= 3 {
		return handleBatchSave(ctx, update, args[1], args[2])
	}

	replyTo := update.EffectiveMessage.ReplyToMessage
	if replyTo == nil || replyTo.Message == nil {
		ctx.Reply(update, ext.ReplyTextString(msgelem.SaveHelpText), nil)
		return dispatcher.EndGroups
	}

	media := replyTo.Media
	supported := mediautil.IsSupported(media)
	if !supported {
		ctx.Reply(update, ext.ReplyTextString("不支持的消息类型"), nil)
		return dispatcher.EndGroups
	}

	msg, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		logger.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}

	genFilename := func() string {
		if len(args) > 1 {
			return args[1]
		}
		filename := tgutil.GenFileNameFromMessage(*replyTo.Message)
		if filename == "" {
			return xid.New().String()
		}
		return filename
	}()
	option := tfile.WithNameIfEmpty(genFilename)
	if len(args) > 1 {
		option = tfile.WithName(genFilename)
	}

	file, err := tfile.FromMedia(media, option)
	if err != nil {
		logger.Errorf("获取文件失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取文件失败: "+err.Error()), nil)
		return dispatcher.EndGroups
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
		return handleBatchSave(ctx, update, args[1], args[2])
	}

	logger := log.FromContext(ctx)
	stor := storage.FromContext(ctx)
	if stor == nil {
		logger.Warn("Context storage is nil")
		ctx.Reply(update, ext.ReplyTextString("未找到存储"), nil)
		return dispatcher.EndGroups
	}
	replyTo := update.EffectiveMessage.ReplyToMessage
	if replyTo == nil || replyTo.Message == nil {
		ctx.Reply(update, ext.ReplyTextString(msgelem.SaveHelpText), nil)
		return dispatcher.EndGroups
	}

	media := replyTo.Media
	supported := mediautil.IsSupported(media)
	if !supported {
		ctx.Reply(update, ext.ReplyTextString("不支持的消息类型"), nil)
		return dispatcher.EndGroups
	}

	msg, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		logger.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}
	genFilename := func() string {
		if len(args) > 1 {
			return args[1]
		}
		filename := tgutil.GenFileNameFromMessage(*replyTo.Message)
		if filename == "" {
			return xid.New().String()
		}
		return filename
	}()
	option := tfile.WithNameIfEmpty(genFilename)
	if len(args) > 1 {
		option = tfile.WithName(genFilename)
	}

	file, err := tfile.FromMedia(media, option)
	if err != nil {
		logger.Errorf("获取文件失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取文件失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	storagePath := stor.JoinStoragePath(file.Name())
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	taskid := xid.New().String()
	task, err := tftask.NewTGFileTask(taskid, injectCtx, file, ctx.Raw, stor, storagePath, tftask.NewProgressTrack(
		msg.ID,
		update.GetUserChat().GetID()))
	if err != nil {
		logger.Errorf("创建任务失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("创建任务失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}

	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("添加任务失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("添加任务失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	text, entities := msgelem.BuildTaskAddedEntities(ctx, file.Name(), core.GetLength(injectCtx))
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:       msg.ID,
		Message:  text,
		Entities: entities,
	})
	return dispatcher.EndGroups
}

func handleBatchSave(ctx *ext.Context, update *ext.Update, chatArg string, msgIdRangeArg string) error {
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

	msgs, err := tgutil.GetMessagesRange(ctx, chatID, int(startID), int(endID))
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("获取消息失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	if len(msgs) == 0 {
		ctx.Reply(update, ext.ReplyTextString("没有找到指定范围内的消息"), nil)
		return dispatcher.EndGroups
	}
	files := make([]tfile.TGFile, 0, len(msgs))
	for _, msg := range msgs {
		media, ok := msg.GetMedia()
		if !ok {
			continue
		}
		supported := mediautil.IsSupported(media)
		if !supported {
			continue
		}
		file, err := tfile.FromMedia(media, tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*msg)))
		if err != nil {
			log.FromContext(ctx).Errorf("获取文件失败: %s", err)
			continue
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		ctx.Reply(update, ext.ReplyTextString("没有找到指定范围内的可保存消息"), nil)
		return dispatcher.EndGroups
	}
	stors := storage.GetUserStorages(ctx, update.GetUserChat().GetID())
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:          replied.ID,
		Message:     fmt.Sprintf("找到 %d 个文件, 请选择存储位置", len(files)),
		ReplyMarkup: msgelem.BuildAddBatchSelectStorageKeyboard(stors, files),
	})
	return dispatcher.EndGroups
}
