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
		logger.L.Panicf("创建正则表达式过滤器失败: %s", err)
	}
	dispatcher.AddHandler(handlers.NewMessage(linkRegexFilter, handleLinkMessage))
	dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("add"), AddToQueue))
	dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("set_default"), setDefaultStorage))
	dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, handleFileMessage))
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
		logger.L.Errorf("未知存储位置哈希: %d", storageNameHash)
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
		logger.L.Errorf("获取记录失败: %s", err)
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
			logger.L.Errorf("更新接收的文件失败: %s", err)
		}
	}
	file, err := FileFromMessage(ctx, record.ChatID, record.MessageID, record.FileName)
	if err != nil {
		logger.L.Errorf("获取消息中的文件失败: %s", err)
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
		UserID:         update.GetUserChat().GetID(),
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
