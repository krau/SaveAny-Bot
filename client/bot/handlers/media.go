package handlers

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tftask"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func handleMediaMessage(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	message := update.EffectiveMessage.Message
	logger.Debugf("Got media: %s", message.Media.TypeName())
	media := message.Media
	supported := mediautil.IsSupported(media)
	if !supported {
		return dispatcher.EndGroups
	}

	msg, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		logger.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}

	genFilename := tgutil.GenFileNameFromMessage(*message)
	if genFilename == "" {
		genFilename = xid.New().String()
	}

	file, err := tfile.FromMedia(media, tfile.WithNameIfEmpty(genFilename))
	if err != nil {
		logger.Errorf("获取文件失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取文件失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	userId := update.GetUserChat().GetID()
	stors := storage.GetUserStorages(ctx, userId)
	req, err := msgelem.BuildSelectStorageMessage(ctx, stors, file, msg.ID)
	if err != nil {
		logger.Errorf("构建存储选择消息失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("构建存储选择消息失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	ctx.EditMessage(update.EffectiveChat().GetID(), req)
	return dispatcher.EndGroups
}

func handleSilentSaveMedia(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	stor := storage.FromContext(ctx)
	if stor == nil {
		logger.Warn("Context storage is nil")
		ctx.Reply(update, ext.ReplyTextString("未找到存储"), nil)
		return dispatcher.EndGroups
	}
	message := update.EffectiveMessage.Message
	logger.Debugf("Got media: %s", message.Media.TypeName())
	chatID := update.EffectiveChat().GetID()
	media := message.Media
	supported := mediautil.IsSupported(media)
	if !supported {
		return dispatcher.EndGroups
	}
	msg, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		logger.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}
	genFilename := tgutil.GenFileNameFromMessage(*message)
	if genFilename == "" {
		genFilename = xid.New().String()
	}

	file, err := tfile.FromMedia(media, tfile.WithNameIfEmpty(genFilename))
	if err != nil {
		logger.Errorf("获取文件失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取文件失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	storagePath := stor.JoinStoragePath(file.Name())
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	taskid := xid.New().String()
	task, err := tftask.NewTGFileTask(taskid, injectCtx, file, ctx.Raw, stor, storagePath,
		tftask.NewProgressTrack(
			msg.ID,
			update.GetUserChat().GetID()))
	if err != nil {
		logger.Errorf("创建任务失败: %s", err)
		ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
			ID:      msg.ID,
			Message: "创建任务失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("添加任务失败: %s", err)
		ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
			ID:      msg.ID,
			Message: "添加任务失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	text, entities := msgelem.BuildTaskAddedEntities(ctx, file.Name(), core.GetLength(injectCtx))
	ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
		ID:       msg.ID,
		Message:  text,
		Entities: entities,
	})

	return dispatcher.EndGroups
}
