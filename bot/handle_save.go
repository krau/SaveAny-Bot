package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func sendSaveHelp(ctx *ext.Context, update *ext.Update) error {
	helpText := `
使用方法:

1. 使用该命令回复要保存的文件, 可选文件名参数.
示例:
/save custom_file_name.mp4

2. 设置默认存储后, 发送 /save <频道ID/用户名> <消息ID范围> 来批量保存文件. 遵从存储规则, 若未匹配到任何规则则使用默认存储.
示例:
/save @moreacg 114-514
	`
	ctx.Reply(update, ext.ReplyTextString(helpText), nil)
	return dispatcher.EndGroups
}

func saveCmd(ctx *ext.Context, update *ext.Update) error {
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) >= 3 {
		return handleBatchSave(ctx, update, args[1:])
	}

	replyToMsgID := func() int {
		res, ok := update.EffectiveMessage.GetReplyTo()
		if !ok || res == nil {
			return 0
		}
		replyHeader, ok := res.(*tg.MessageReplyHeader)
		if !ok {
			return 0
		}
		replyToMsgID, ok := replyHeader.GetReplyToMsgID()
		if !ok {
			return 0
		}
		return replyToMsgID
	}()
	if replyToMsgID == 0 {
		return sendSaveHelp(ctx, update)
	}

	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		common.Log.Errorf("获取用户失败: %s", err)
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
		common.Log.Errorf("获取消息失败: %s", err)
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
		common.Log.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}

	cmdText := update.EffectiveMessage.Text
	customFileName := strings.TrimSpace(strings.TrimPrefix(cmdText, "/save"))

	file, err := FileFromMessage(ctx, update.EffectiveChat().GetID(), msg.ID, customFileName)
	if err != nil {
		common.Log.Errorf("获取文件失败: %s", err)
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: fmt.Sprintf("获取文件失败: %s", err),
			ID:      replied.ID,
		})
		return dispatcher.EndGroups
	}

	if file.FileName == "" {
		file.FileName = GenFileNameFromMessage(*msg, file)
	}
	receivedFile := &dao.ReceivedFile{
		Processing:     false,
		FileName:       file.FileName,
		ChatID:         update.EffectiveChat().GetID(),
		MessageID:      replyToMsgID,
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
	}

	if err := dao.SaveReceivedFile(receivedFile); err != nil {
		common.Log.Errorf("保存接收的文件失败: %s", err)
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: fmt.Sprintf("保存接收的文件失败: %s", err),
			ID:      replied.ID,
		}); err != nil {
			common.Log.Errorf("编辑消息失败: %s", err)
		}
		return dispatcher.EndGroups
	}
	if !user.Silent || user.DefaultStorage == "" {
		return ProvideSelectMessage(ctx, update, file.FileName, update.EffectiveChat().GetID(), msg.ID, replied.ID)
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

func handleBatchSave(ctx *ext.Context, update *ext.Update, args []string) error {
	// args: [0] = @channel, [1] = 114-514
	chatArg := args[0]
	var chatID int64
	var err error
	msgIdSlice := strings.Split(args[1], "-")
	if len(msgIdSlice) != 2 {
		ctx.Reply(update, ext.ReplyTextString("无效的消息ID范围"), nil)
		return dispatcher.EndGroups
	}
	minMsgID, minerr := strconv.ParseInt(msgIdSlice[0], 10, 64)
	maxMsgID, maxerr := strconv.ParseInt(msgIdSlice[1], 10, 64)
	if minerr != nil || maxerr != nil {
		ctx.Reply(update, ext.ReplyTextString("无效的消息ID范围"), nil)
		return dispatcher.EndGroups
	}
	if minMsgID > maxMsgID || minMsgID <= 0 || maxMsgID <= 0 {
		ctx.Reply(update, ext.ReplyTextString("无效的消息ID范围"), nil)
		return dispatcher.EndGroups
	}
	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		common.Log.Errorf("获取用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}
	if user.DefaultStorage == "" {
		ctx.Reply(update, ext.ReplyTextString("请先设置默认存储"), nil)
		return dispatcher.EndGroups
	}
	storages := storage.GetUserStorages(user.ChatID)
	if len(storages) == 0 {
		ctx.Reply(update, ext.ReplyTextString("无可用的存储"), nil)
		return dispatcher.EndGroups
	}

	if strings.HasPrefix(chatArg, "@") {
		chatUsername := strings.TrimPrefix(chatArg, "@")
		chat, err := ctx.ResolveUsername(chatUsername)
		if err != nil {
			common.Log.Errorf("解析频道用户名失败: %s", err)
			ctx.Reply(update, ext.ReplyTextString("解析频道用户名失败"), nil)
			return dispatcher.EndGroups
		}
		if chat == nil {
			ctx.Reply(update, ext.ReplyTextString("无法找到聊天"), nil)
			return dispatcher.EndGroups
		}
		chatID = chat.GetID()
	} else {
		chatID, err = strconv.ParseInt(chatArg, 10, 64)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString("无效的频道ID或用户名"), nil)
			return dispatcher.EndGroups
		}
	}
	if chatID == 0 {
		ctx.Reply(update, ext.ReplyTextString("无效的频道ID或用户名"), nil)
		return dispatcher.EndGroups
	}

	replied, err := ctx.Reply(update, ext.ReplyTextString("正在批量保存..."), nil)
	if err != nil {
		common.Log.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}

	total := maxMsgID - minMsgID + 1
	successadd := 0
	failedGetFile := 0
	failedGetMsg := 0
	failedSaveDB := 0
	for i := minMsgID; i <= maxMsgID; i++ {
		file, err := FileFromMessage(ctx, chatID, int(i), "")
		if err != nil {
			common.Log.Errorf("获取文件失败: %s", err)
			failedGetFile++
			continue
		}
		if file.FileName == "" {
			message, err := GetTGMessage(ctx, chatID, int(i))
			if err != nil {
				common.Log.Errorf("获取消息失败: %s", err)
				failedGetMsg++
				continue
			}
			file.FileName = GenFileNameFromMessage(*message, file)
		}
		receivedFile := &dao.ReceivedFile{
			Processing:     false,
			FileName:       file.FileName,
			ChatID:         chatID,
			MessageID:      int(i),
			ReplyChatID:    update.GetUserChat().GetID(),
			ReplyMessageID: 0,
		}
		if err := dao.SaveReceivedFile(receivedFile); err != nil {
			common.Log.Errorf("保存接收的文件失败: %s", err)
			failedSaveDB++
			continue
		}
		task := &types.Task{
			Ctx:            ctx,
			Status:         types.Pending,
			File:           file,
			StorageName:    user.DefaultStorage,
			FileChatID:     chatID,
			FileMessageID:  int(i),
			UserID:         user.ChatID,
			ReplyMessageID: 0,
			ReplyChatID:    update.GetUserChat().GetID(),
		}
		queue.AddTask(task)
		successadd++
	}
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("批量添加任务完成\n成功添加: %d/%d\n获取文件失败: %d\n获取消息失败: %d\n保存数据库失败: %d", successadd, total, failedGetFile, failedGetMsg, failedSaveDB),
		ID:      replied.ID,
	})
	return dispatcher.EndGroups
}
