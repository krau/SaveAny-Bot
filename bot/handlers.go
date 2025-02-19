package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func RegisterHandlers(dispatcher dispatcher.Dispatcher) {
	dispatcher.AddHandler(handlers.NewMessage(filters.Message.All, checkPermission))
	dispatcher.AddHandler(handlers.NewCommand("start", start))
	dispatcher.AddHandler(handlers.NewCommand("help", help))
	dispatcher.AddHandler(handlers.NewCommand("silent", silent))
	dispatcher.AddHandler(handlers.NewCommand("storage", storageCmd))
	dispatcher.AddHandler(handlers.NewCommand("save", saveCmd))
	linkRegexFilter, err := filters.Message.Regex(linkRegexString)
	if err != nil {
		logger.L.Panicf("Failed to create regex filter: %s", err)
	}
	dispatcher.AddHandler(handlers.NewMessage(linkRegexFilter, handleLinkMessage))
	dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("add"), AddToQueue))
	dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("set_default"), setDefaultStorage))
	dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, handleFileMessage))
}

const noPermissionText string = `
您不在白名单中, 无法使用此 Bot.
您可以部署自己的实例: https://github.com/krau/SaveAny-Bot
`

func checkPermission(ctx *ext.Context, update *ext.Update) error {
	userID := update.GetUserChat().GetID()
	if !slice.Contain(config.Cfg.Telegram.Admins, userID) {
		ctx.Reply(update, ext.ReplyTextString(noPermissionText), nil)
		return dispatcher.EndGroups
	}
	return dispatcher.ContinueGroups
}

func start(ctx *ext.Context, update *ext.Update) error {
	if err := dao.CreateUser(update.GetUserChat().GetID()); err != nil {
		logger.L.Errorf("Failed to create user: %s", err)
		return dispatcher.EndGroups
	}
	return help(ctx, update)
}

const helpText string = `
Save Any Bot - 转存你的 Telegram 文件
命令:
/start - 开始使用
/help - 显示帮助
/silent - 开关静默模式
/storage - 设置默认存储位置
/save [自定义文件名] - 保存文件

静默模式: 开启后 Bot 直接保存到收到的文件到默认位置, 不再询问

默认存储位置: 在静默模式下保存到的位置

向 Bot 发送(转发)文件, 或发送一个公开频道的消息链接以保存文件
`

func help(ctx *ext.Context, update *ext.Update) error {
	ctx.Reply(update, ext.ReplyTextString(helpText), nil)
	return dispatcher.EndGroups
}

func silent(ctx *ext.Context, update *ext.Update) error {
	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		logger.L.Errorf("Failed to get user: %s", err)
		return dispatcher.EndGroups
	}
	user.Silent = !user.Silent
	if err := dao.UpdateUser(user); err != nil {
		logger.L.Errorf("Failed to update user: %s", err)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("已%s静默模式", map[bool]string{true: "开启", false: "关闭"}[user.Silent])), nil)
	return dispatcher.EndGroups
}

func saveCmd(ctx *ext.Context, update *ext.Update) error {
	res, ok := update.EffectiveMessage.GetReplyTo()
	if !ok || res == nil {
		ctx.Reply(update, ext.ReplyTextString("请回复要保存的文件"), nil)
		return dispatcher.EndGroups
	}
	replyHeader, ok := res.(*tg.MessageReplyHeader)
	if !ok {
		ctx.Reply(update, ext.ReplyTextString("请回复要保存的文件"), nil)
		return dispatcher.EndGroups
	}
	replyToMsgID, ok := replyHeader.GetReplyToMsgID()
	if !ok {
		ctx.Reply(update, ext.ReplyTextString("请回复要保存的文件"), nil)
		return dispatcher.EndGroups
	}

	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		logger.L.Errorf("Failed to get user: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}

	storages := storage.GetUserStorages(user.ChatID)

	if len(storages) == 0 {
		ctx.Reply(update, ext.ReplyTextString("无可用的存储"), nil)
		return dispatcher.EndGroups
	}

	msg, err := GetTGMessage(ctx, update.EffectiveChat().GetID(), replyToMsgID)
	if err != nil {
		logger.L.Errorf("Failed to get message: %s", err)
		ctx.Reply(update, ext.ReplyTextString("无法获取消息"), nil)
		return dispatcher.EndGroups
	}

	supported, _ := supportedMediaFilter(msg)
	if !supported {
		ctx.Reply(update, ext.ReplyTextString("不支持的消息类型或消息中没有文件"), nil)
		return dispatcher.EndGroups
	}

	replied, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		logger.L.Errorf("Failed to reply: %s", err)
		return dispatcher.EndGroups
	}

	cmdText := update.EffectiveMessage.Text
	customFileName := strings.TrimSpace(strings.TrimPrefix(cmdText, "/save"))

	file, err := FileFromMessage(ctx, update.EffectiveChat().GetID(), msg.ID, customFileName)
	if err != nil {
		logger.L.Errorf("Failed to get file from message: %s", err)
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: fmt.Sprintf("获取文件失败: %s", err),
			ID:      replied.ID,
		})
		return dispatcher.EndGroups
	}

	// TODO: better file name
	if file.FileName == "" {
		file.FileName = fmt.Sprintf("%d_%d_%s", update.EffectiveChat().GetID(), replyToMsgID, file.Hash())
	}
	receivedFile := &types.ReceivedFile{
		Processing:     false,
		FileName:       file.FileName,
		ChatID:         update.EffectiveChat().GetID(),
		MessageID:      replyToMsgID,
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
	}

	if err := dao.SaveReceivedFile(receivedFile); err != nil {
		logger.L.Errorf("Failed to save received file: %s", err)
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: fmt.Sprintf("Failed to save received file: %s", err),
			ID:      replied.ID,
		}); err != nil {
			logger.L.Errorf("Failed to edit message: %s", err)
		}
		return dispatcher.EndGroups
	}
	if !user.Silent || user.DefaultStorage == "" {
		return ProvideSelectMessage(ctx, update, file, update.EffectiveChat().GetID(), msg.ID, replied.ID)
	}
	return HandleSilentAddTask(ctx, update, user, &types.Task{
		Ctx:            ctx,
		Status:         types.Pending,
		File:           file,
		StorageName:    user.DefaultStorage,
		FileChatID:     update.EffectiveChat().GetID(),
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
		FileMessageID:  msg.ID,
		UserID:         user.ChatID,
	})
}

func storageCmd(ctx *ext.Context, update *ext.Update) error {
	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		logger.L.Errorf("Failed to get user: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}
	storages := storage.GetUserStorages(user.ChatID)
	if len(storages) == 0 {
		ctx.Reply(update, ext.ReplyTextString("无可用的存储"), nil)
		return dispatcher.EndGroups
	}

	ctx.Reply(update, ext.ReplyTextString("请选择要设为默认的存储位置"), &ext.ReplyOpts{
		Markup: getSetDefaultStorageMarkup(user.ChatID, storages),
	})

	return dispatcher.EndGroups
}

func setDefaultStorage(ctx *ext.Context, update *ext.Update) error {
	args := strings.Split(string(update.CallbackQuery.Data), " ")
	userID, _ := strconv.Atoi(args[1])
	storageNameHash := args[2]
	if userID != int(update.CallbackQuery.GetUserID()) {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "你没有权限",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	storageName := storageHashName[storageNameHash]
	selectedStorage, err := storage.GetStorageByName(storageName)

	if err != nil {
		logger.L.Errorf("failed to get storage: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "获取指定存储失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	user, err := dao.GetUserByChatID(int64(userID))
	if err != nil {
		logger.L.Errorf("Failed to get user: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "获取用户失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	user.DefaultStorage = storageName
	if err := dao.UpdateUser(user); err != nil {
		logger.L.Errorf("Failed to update user: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "更新用户失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("已将 %s (%s) 设为默认存储位置", selectedStorage.Name(), selectedStorage.Type()),
		ID:      update.CallbackQuery.GetMsgID(),
	})
	return dispatcher.EndGroups
}

func handleFileMessage(ctx *ext.Context, update *ext.Update) error {
	logger.L.Trace("Got media: ", update.EffectiveMessage.Media.TypeName())
	supported, err := supportedMediaFilter(update.EffectiveMessage.Message)
	if err != nil {
		return err
	}
	if !supported {
		return dispatcher.EndGroups
	}

	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		logger.L.Errorf("Failed to get user: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}
	storages := storage.GetUserStorages(user.ChatID)
	if len(storages) == 0 {
		ctx.Reply(update, ext.ReplyTextString("无可用的存储"), nil)
		return dispatcher.EndGroups
	}

	msg, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		logger.L.Errorf("Failed to reply: %s", err)
		return dispatcher.EndGroups
	}
	media := update.EffectiveMessage.Media
	file, err := FileFromMedia(media, "")
	if err != nil {
		logger.L.Errorf("Failed to get file from media: %s", err)
		ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("获取文件失败: %s", err)), nil)
		return dispatcher.EndGroups
	}
	if file.FileName == "" {
		file.FileName = fmt.Sprintf("%d_%d_%s", update.EffectiveChat().GetID(), update.EffectiveMessage.ID, file.Hash())
	}

	if err := dao.SaveReceivedFile(&types.ReceivedFile{
		Processing:     false,
		FileName:       file.FileName,
		ChatID:         update.EffectiveChat().GetID(),
		MessageID:      update.EffectiveMessage.ID,
		ReplyMessageID: msg.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
	}); err != nil {
		logger.L.Errorf("Failed to add received file: %s", err)
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: fmt.Sprintf("Failed to add received file: %s", err),
			ID:      msg.ID,
		}); err != nil {
			logger.L.Errorf("Failed to edit message: %s", err)
		}
		return dispatcher.EndGroups
	}

	if !user.Silent || user.DefaultStorage == "" {
		return ProvideSelectMessage(ctx, update, file, update.EffectiveChat().GetID(), update.EffectiveMessage.ID, msg.ID)
	}
	return HandleSilentAddTask(ctx, update, user, &types.Task{
		Ctx:            ctx,
		Status:         types.Pending,
		File:           file,
		StorageName:    user.DefaultStorage,
		FileChatID:     update.EffectiveChat().GetID(),
		ReplyMessageID: msg.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
		FileMessageID:  update.EffectiveMessage.ID,
		UserID:         user.ChatID,
	})
}

func AddToQueue(ctx *ext.Context, update *ext.Update) error {
	if !slice.Contain(config.Cfg.GetUsersID(), update.CallbackQuery.UserID) {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "你没有权限",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	args := strings.Split(string(update.CallbackQuery.Data), " ")
	fileChatID, _ := strconv.Atoi(args[1])
	fileMessageID, _ := strconv.Atoi(args[2])
	storageNameHash := args[3]
	storageName := storageHashName[storageNameHash]
	if storageName == "" {
		logger.L.Errorf("Unknown storage name hash: %d", storageNameHash)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "未知存储位置",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	logger.L.Tracef("Got add to queue: chatID: %d, messageID: %d, storage: %s", fileChatID, fileMessageID, storageName)
	record, err := dao.GetReceivedFileByChatAndMessageID(int64(fileChatID), fileMessageID)
	if err != nil {
		logger.L.Errorf("Failed to get received file: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "查询记录失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}
	if update.CallbackQuery.MsgID != record.ReplyMessageID {
		record.ReplyMessageID = update.CallbackQuery.MsgID
		if err := dao.SaveReceivedFile(record); err != nil {
			logger.L.Errorf("Failed to update received file: %s", err)
		}
	}
	file, err := FileFromMessage(ctx, record.ChatID, record.MessageID, record.FileName)
	if err != nil {
		logger.L.Errorf("Failed to get file from message: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   fmt.Sprintf("获取消息中的文件失败: %s", err),
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}

	queue.AddTask(types.Task{
		Ctx:            ctx,
		Status:         types.Pending,
		File:           file,
		StorageName:    storageName,
		FileChatID:     record.ChatID,
		ReplyMessageID: record.ReplyMessageID,
		FileMessageID:  record.MessageID,
		ReplyChatID:    record.ReplyChatID,
		UserID:         update.EffectiveUser().GetID(),
	})

	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	text := fmt.Sprintf("已添加到任务队列\n文件名: %s\n当前排队任务数: %d", record.FileName, queue.Len())
	if err := styling.Perform(&entityBuilder,
		styling.Plain("已添加到任务队列\n文件名: "),
		styling.Code(record.FileName),
		styling.Plain("\n当前排队任务数: "),
		styling.Bold(strconv.Itoa(queue.Len())),
	); err != nil {
		logger.L.Errorf("Failed to build entity: %s", err)
	} else {
		text, entities = entityBuilder.Complete()
	}

	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		Message:  text,
		Entities: entities,
		ID:       record.ReplyMessageID,
	})
	return dispatcher.EndGroups
}
