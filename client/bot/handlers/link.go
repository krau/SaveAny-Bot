package handlers

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/batchtftask"
	"github.com/krau/SaveAny-Bot/core/tftask"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func handleMessageLink(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	msgLinks := re.TgMessageLinkRegexp.FindAllString(update.EffectiveMessage.GetMessage(), -1)
	if len(msgLinks) == 0 {
		logger.Warn("no matched message links but called handleMessageLink")
		return dispatcher.ContinueGroups
	}
	replied, err := ctx.Reply(update, ext.ReplyTextString("正在获取消息..."), nil)
	if err != nil {
		logger.Errorf("failed to reply: %s", err)
		return dispatcher.EndGroups
	}
	editReplied := func(text string, markup tg.ReplyMarkupClass) {
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:          replied.ID,
			Message:     text,
			ReplyMarkup: markup,
		}); err != nil {
			logger.Errorf("failed to edit message: %s", err)
		}
	}

	files := make([]tfile.TGFile, 0, len(msgLinks))
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
		file, err := tfile.FromMedia(media, tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*msg)))
		if err != nil {
			logger.Errorf("failed to create file from media: %s", err)
			continue
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		editReplied("没有找到可保存的文件", nil)
		return dispatcher.EndGroups
	}
	userId := update.GetUserChat().GetID()
	stors := storage.GetUserStorages(ctx, userId)
	if len(files) == 1 {
		req, err := msgelem.BuildAddOneSelectStorageMessage(ctx, stors, files[0], replied.ID)
		if err != nil {
			logger.Errorf("构建存储选择消息失败: %s", err)
			editReplied("构建存储选择消息失败: "+err.Error(), nil)
			return dispatcher.EndGroups
		}
		ctx.EditMessage(update.EffectiveChat().GetID(), req)
		return dispatcher.EndGroups
	}
	editReplied(fmt.Sprintf("找到 %d 个文件, 请选择存储位置", len(files)),
		msgelem.BuildAddBatchSelectStorageKeyboard(stors, files))
	return dispatcher.EndGroups
}

func handleSilentSaveLink(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	stor := storage.FromContext(ctx)
	if stor == nil {
		logger.Warn("Context storage is nil")
		ctx.Reply(update, ext.ReplyTextString("未找到存储"), nil)
		return dispatcher.EndGroups
	}
	msgLinks := re.TgMessageLinkRegexp.FindAllString(update.EffectiveMessage.GetMessage(), -1)
	if len(msgLinks) == 0 {
		logger.Warn("no matched message links but called handleMessageLink")
		return dispatcher.ContinueGroups
	}
	replied, err := ctx.Reply(update, ext.ReplyTextString("正在获取消息..."), nil)
	if err != nil {
		logger.Errorf("failed to reply: %s", err)
		return dispatcher.EndGroups
	}
	editReplied := func(text string, markup tg.ReplyMarkupClass) {
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:          replied.ID,
			Message:     text,
			ReplyMarkup: markup,
		}); err != nil {
			logger.Errorf("failed to edit message: %s", err)
		}
	}

	files := make([]tfile.TGFile, 0, len(msgLinks))
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
		file, err := tfile.FromMedia(media, tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(*msg)))
		if err != nil {
			logger.Errorf("failed to create file from media: %s", err)
			continue
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		editReplied("没有找到可保存的文件", nil)
		return dispatcher.EndGroups
	}
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	taskid := xid.New().String()
	if len(files) == 1 {
		file := files[0]
		task, err := tftask.NewTGFileTask(taskid, injectCtx, file, ctx.Raw, stor, stor.JoinStoragePath(file.Name()), tftask.NewProgressTrack(replied.ID, update.EffectiveChat().GetID()))
		if err != nil {
			logger.Errorf("Failed to create task: %s", err)
			editReplied("任务创建失败: "+err.Error(), nil)
			return dispatcher.EndGroups
		}
		if err := core.AddTask(injectCtx, task); err != nil {
			logger.Errorf("Failed to add task: %s", err)
			editReplied("批量任务添加失败: "+err.Error(), nil)
			return dispatcher.EndGroups
		}
		text, entities := msgelem.BuildTaskAddedEntities(ctx, file.Name(), core.GetLength(injectCtx))
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:       replied.ID,
			Message:  text,
			Entities: entities,
		})
		return dispatcher.EndGroups
	}
	elems := make([]batchtftask.TaskElement, 0, len(files))
	for _, file := range files {
		storPath := stor.JoinStoragePath(file.Name())
		elem, err := batchtftask.NewTaskElement(stor, storPath, file)
		if err != nil {
			logger.Errorf("Failed to create task element: %s", err)
			editReplied("任务创建失败: "+err.Error(), nil)
			return dispatcher.EndGroups
		}
		elems = append(elems, *elem)
	}

	task := batchtftask.NewBatchTGFileTask(taskid, injectCtx, elems, ctx.Raw, batchtftask.NewProgressTracker(replied.ID, update.EffectiveChat().GetID()), true)
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("Failed to add batch task: %s", err)
		editReplied("批量任务添加失败: "+err.Error(), nil)
		return dispatcher.EndGroups
	}
	editReplied(fmt.Sprintf("已添加批量任务, 共 %d 个文件", len(files)), nil)
	return dispatcher.EndGroups
}
