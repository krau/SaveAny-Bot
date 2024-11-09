package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/gookit/goutil/maputil"
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
	dispatcher.AddHandler(handlers.NewAnyUpdate(checkPermission))
	dispatcher.AddHandler(handlers.NewCommand("start", start))
	dispatcher.AddHandler(handlers.NewCommand("help", help))
	dispatcher.AddHandler(handlers.NewCommand("silent", silent))
	dispatcher.AddHandler(handlers.NewCommand("storage", setDefaultStorage))
	dispatcher.AddHandler(handlers.NewCommand("save", saveCmd))
	dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("add"), AddToQueue))
	dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, handleFileMessage))
}

const noPermissionText string = `
本 Bot 仅限个人使用.
您可以部署自己的实例: https://github.com/krau/SaveAny-Bot
`

func checkPermission(ctx *ext.Context, update *ext.Update) error {
	userID := update.GetUserChat().GetID()
	if !slice.Contain(config.Cfg.Telegram.Admins, userID) {
		ctx.Reply(update, noPermissionText, nil)
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
SaveAny Bot - 转存你的 Telegram 文件
命令:
/start - 开始使用
/help - 显示帮助
/silent - 静默模式
/storage - 设置默认存储位置
/save - 保存文件

静默模式: 开启后 Bot 直接保存到收到的文件到默认位置, 不再询问
`

func help(ctx *ext.Context, update *ext.Update) error {
	ctx.Reply(update, helpText, nil)
	return dispatcher.EndGroups
}

func silent(ctx *ext.Context, update *ext.Update) error {
	user, err := dao.GetUserByUserID(update.GetUserChat().GetID())
	if err != nil {
		logger.L.Errorf("Failed to get user: %s", err)
		return dispatcher.EndGroups
	}
	user.Silent = !user.Silent
	if err := dao.UpdateUser(user); err != nil {
		logger.L.Errorf("Failed to update user: %s", err)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, fmt.Sprintf("已%s静默模式", func() string {
		if user.Silent {
			return "开启"
		}
		return "关闭"
	}()), nil)
	return dispatcher.EndGroups
}

func setDefaultStorage(ctx *ext.Context, update *ext.Update) error {
	if len(storage.Storages) == 0 {
		ctx.Reply(update, "未配置存储", nil)
		return dispatcher.EndGroups
	}
	args := strings.Split(update.EffectiveMessage.Text, " ")
	avaliableStorages := maputil.Keys(storage.Storages)
	if len(args) < 2 {
		text := []styling.StyledTextOption{
			styling.Plain("请提供存储位置名称, 可用项:"),
		}
		for _, name := range avaliableStorages {
			text = append(text, styling.Plain("\n"))
			text = append(text, styling.Code(name))
		}
		ctx.Reply(update, text, nil)
		return dispatcher.EndGroups
	}
	storageName := args[1]
	if !slice.Contain(avaliableStorages, storageName) {
		ctx.Reply(update, "存储位置不存在", nil)
		return dispatcher.EndGroups
	}
	user, err := dao.GetUserByUserID(update.GetUserChat().GetID())
	if err != nil {
		logger.L.Errorf("Failed to get user: %s", err)
		return dispatcher.EndGroups
	}
	user.DefaultStorage = storageName
	if err := dao.UpdateUser(user); err != nil {
		logger.L.Errorf("Failed to update user: %s", err)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, fmt.Sprintf("已设置默认存储位置为 %s", storageName), nil)
	return dispatcher.EndGroups
}

func saveCmd(ctx *ext.Context, update *ext.Update) error {
	// TODO: Implement save command
	return dispatcher.EndGroups
}

func handleFileMessage(ctx *ext.Context, update *ext.Update) error {
	logger.L.Trace("Got media: ", update.EffectiveMessage.Media.TypeName())
	supported, err := supportedMediaFilter(update.EffectiveMessage)
	if err != nil {
		return err
	}
	if !supported {
		return dispatcher.EndGroups
	}

	user, err := dao.GetUserByUserID(update.GetUserChat().GetID())
	if err != nil {
		logger.L.Errorf("Failed to get user: %s", err)
		return dispatcher.EndGroups
	}

	msg, err := ctx.Reply(update, "正在获取文件信息...", nil)
	if err != nil {
		logger.L.Errorf("Failed to reply: %s", err)
		return dispatcher.EndGroups
	}
	media := update.EffectiveMessage.Media
	file, err := FileFromMedia(media)
	if err != nil {
		logger.L.Errorf("Failed to get file from media: %s", err)
		ctx.Reply(update, "无法获取文件", nil)
		return dispatcher.EndGroups
	}
	if file.FileName == "" {
		ctx.Reply(update, "无法获取文件名", nil)
		return dispatcher.EndGroups
	}

	if err := dao.AddReceivedFile(&types.ReceivedFile{
		Processing:     false,
		FileName:       file.FileName,
		ChatID:         update.EffectiveChat().GetID(),
		MessageID:      update.EffectiveMessage.ID,
		ReplyMessageID: msg.ID,
	}); err != nil {
		logger.L.Errorf("Failed to add received file: %s", err)
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: "无法保存文件",
			ID:      msg.ID,
		}); err != nil {
			logger.L.Errorf("Failed to edit message: %s", err)
		}

		return dispatcher.EndGroups
	}

	if !user.Silent {
		text := "请选择存储位置"
		_, err = ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message:     text,
			ReplyMarkup: getAddTaskMarkup(update.EffectiveMessage.ID),
			ID:          msg.ID,
		})
		if err != nil {
			logger.L.Errorf("Failed to edit message: %s", err)
		}
		return dispatcher.EndGroups
	}

	if user.DefaultStorage == "" {
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: "请先使用 /storage 设置默认存储位置",
			ID:      msg.ID,
		})
		return dispatcher.EndGroups
	}

	queue.AddTask(types.Task{
		Ctx:            ctx,
		Status:         types.Pending,
		File:           file,
		Storage:        types.StorageType(user.DefaultStorage),
		ChatID:         update.EffectiveChat().GetID(),
		ReplyMessageID: msg.ID,
		MessageID:      update.EffectiveMessage.ID,
	})

	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("已添加到队列: %s\n当前排队任务数: %d", file.FileName, queue.Len()),
		ID:      msg.ID,
	})
	return dispatcher.EndGroups
}

func AddToQueue(ctx *ext.Context, update *ext.Update) error {
	args := strings.Split(string(update.CallbackQuery.Data), " ")
	messageID, _ := strconv.Atoi(args[1])
	logger.L.Trace("Got add to queue: chatID: %d, messageID: %d, storage: %s", update.EffectiveChat().GetID(), messageID, args[2])
	record, err := dao.GetReceivedFileByChatAndMessageID(update.EffectiveChat().GetID(), messageID)
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
	file, err := FileFromMessage(ctx, Client, record.ChatID, record.MessageID)
	if err != nil {
		logger.L.Errorf("Failed to get file from message: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "获取消息文件失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}

	queue.AddTask(types.Task{
		Ctx:            ctx,
		Status:         types.Pending,
		File:           file,
		Storage:        types.StorageType(args[2]),
		ChatID:         record.ChatID,
		ReplyMessageID: record.ReplyMessageID,
		MessageID:      record.MessageID,
	})
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("已添加到队列: %s\n当前排队任务数: %d", record.FileName, queue.Len()),
		ID:      record.ReplyMessageID,
	})
	return dispatcher.EndGroups
}
