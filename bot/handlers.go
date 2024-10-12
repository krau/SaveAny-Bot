package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gookit/goutil/maputil"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/model"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
	"github.com/mymmrac/telego/telegoutil"
)

func Start(message *telegram.NewMessage) error {
	if err := dao.CreateUser(message.ChatID()); err != nil {
		logger.L.Errorf("Failed to create user: %s", err)
		return err
	}
	return Help(message)
}

func Help(message *telegram.NewMessage) error {
	helpText := `
SaveAny Bot - 转存你的 Telegram 文件
命令:
/start - 开始使用
/help - 显示帮助
/silent - 静默模式
/storage - 设置默认存储位置
/save - 保存文件

静默模式: 开启后 Bot 直接保存到收到的文件到默认位置, 不再询问
	`
	if _, err := message.Reply(helpText); err != nil {
		logger.L.Errorf("Failed to send help message: %s", err)
		return err
	}
	return nil
}

func ChangeSilentMode(message *telegram.NewMessage) error {
	user, err := dao.GetUserByUserID(message.ChatID())
	if err != nil {
		logger.L.Error(err)
		return err
	}
	user.Silent = !user.Silent
	err = dao.UpdateUser(user)
	if err != nil {
		logger.L.Error(err)
		return err
	}
	if _, err := message.Reply(fmt.Sprintf("已%s静默模式", map[bool]string{true: "开启", false: "关闭"}[user.Silent])); err != nil {
		return err
	}
	return nil
}

func SetDefaultStorage(message *telegram.NewMessage) error {
	if len(storage.Storages) == 0 {
		message.Reply("当前无可用存储端, 请检查配置.")
		return nil
	}
	_, _, args := telegoutil.ParseCommand(message.Text())
	availableStorages := maputil.Keys(storage.Storages)
	if len(args) == 0 {
		text := "请提供存储位置名称, 可用项:"
		for _, name := range availableStorages {
			text += fmt.Sprintf("\n`%s`", name)
		}
		text += fmt.Sprintf("\n`all`")
		message.Reply(text, telegram.SendOptions{ParseMode: telegram.MarkDown})
		return nil
	}
	storageName := args[0]
	if !slice.Contain(availableStorages, storageName) {
		message.Reply("参数错误")
		return nil
	}
	user, err := dao.GetUserByUserID(message.ChatID())
	if err != nil {
		logger.L.Error(err)
		return err
	}
	user.DefaultStorage = storageName
	err = dao.UpdateUser(user)
	if err != nil {
		logger.L.Error(err)
		return err
	}
	if _, err := message.Reply(fmt.Sprintf("已设置默认存储位置为: %s", storageName)); err != nil {
		return err
	}
	return nil
}

func SaveCmd(message *telegram.NewMessage) error {
	targetMessage, err := message.GetReplyMessage()
	if err != nil {
		message.Reply("请回复要保存的文件")
		return nil
	}
	if !targetMessage.IsMedia() {
		message.Reply("回复的消息不包含文件")
		return nil
	}

	msg, err := targetMessage.Reply("正在获取文件信息...")
	if err != nil {
		logger.L.Error(err)
		message.Reply("获取文件信息失败")
		return err
	}

	_, _, _, fileName, err := telegram.GetFileLocation(targetMessage.Media())
	if err != nil {
		logger.L.Error(err)
		targetMessage.Reply("获取文件信息失败")
		return err
	}
	if fileName == "" {
		logger.L.Error("Empty file name")
		targetMessage.Reply("文件名为空")
		return nil
	}

	if err := dao.AddReceivedFile(&model.ReceivedFile{
		Processing:     false,
		FileName:       fileName,
		ChatID:         targetMessage.ChatID(),
		MessageID:      targetMessage.Message.ID,
		ReplyMessageID: msg.ID,
	}); err != nil {
		logger.L.Error(err)
		msg.Edit("保存文件信息失败")
		return err
	}

	user, err := dao.GetUserByUserID(message.ChatID())
	if err != nil {
		logger.L.Error(err)
		msg.Edit("获取用户信息失败")
		return err
	}

	if !user.Silent {
		msg.Edit("请选择要保存的位置:", telegram.SendOptions{
			ReplyMarkup: AddTaskReplyMarkup(targetMessage.Message.ID),
		})
		return nil
	}

	if user.DefaultStorage == "" {
		msg.Edit("请先使用 /storage 命令设置默认存储位置, 或者关闭静默模式")
		return nil
	}

	queue.AddTask(types.Task{
		Ctx:            context.TODO(),
		Status:         types.Pending,
		FileName:       fileName,
		Storage:        types.StorageType(user.DefaultStorage),
		ChatID:         targetMessage.ChatID(),
		MessageID:      targetMessage.Message.ID,
		ReplyMessageID: msg.ID,
	})

	msg.Edit(fmt.Sprintf("已添加到队列: %s\n当前排队任务数: %d", fileName, queue.Len()))

	return nil
}

func HandleFileMessage(message *telegram.NewMessage) error {
	if !message.IsMedia() {
		return nil
	}

	user, err := dao.GetUserByUserID(message.ChatID())
	if err != nil {
		logger.L.Error(err)
		return nil
	}

	msg, err := message.Reply("正在获取文件信息...")
	if err != nil {
		logger.L.Error(err)
		return err
	}

	_, _, _, fileName, err := telegram.GetFileLocation(message.Media())
	if err != nil {
		logger.L.Error(err)
		message.Reply("获取文件信息失败")
		return err
	}
	if fileName == "" {
		logger.L.Error("Empty file name")
		message.Reply("文件名为空")
		return nil
	}

	if err := dao.AddReceivedFile(&model.ReceivedFile{
		Processing:     false,
		FileName:       fileName,
		ChatID:         message.ChatID(),
		MessageID:      message.Message.ID,
		ReplyMessageID: msg.ID,
	}); err != nil {
		logger.L.Error(err)
		msg.Edit("保存文件信息失败")
		return err
	}

	if !user.Silent {
		msg.Edit("请选择要保存的位置:", telegram.SendOptions{
			ReplyMarkup: AddTaskReplyMarkup(message.Message.ID),
		})
		return nil
	}

	if user.DefaultStorage == "" {
		msg.Edit("请先使用 /storage 命令设置默认存储位置, 或者关闭静默模式")
		return nil
	}

	queue.AddTask(types.Task{
		Ctx:            context.TODO(),
		Status:         types.Pending,
		FileName:       fileName,
		Storage:        types.StorageType(user.DefaultStorage),
		ChatID:         message.ChatID(),
		MessageID:      message.Message.ID,
		ReplyMessageID: msg.ID,
	})

	msg.Edit(fmt.Sprintf("已添加到队列: %s\n当前排队任务数: %d", fileName, queue.Len()))
	return nil
}

func AddToQueue(query *telegram.CallbackQuery) error {
	args := strings.Split(query.DataString(), " ")
	messageID, _ := strconv.Atoi(args[1])
	logger.L.Debug(query.ChatID, messageID)
	receivedFile, err := dao.GetReceivedFileByChatAndMessageID(query.ChatID, int32(messageID))
	if err != nil {
		logger.L.Error(err)
		query.Answer("获取文件信息失败", &telegram.CallbackOptions{
			Alert:     true,
			CacheTime: 5,
		})
		return err
	}
	queue.AddTask(types.Task{
		Ctx:            context.TODO(),
		Status:         types.Pending,
		FileName:       receivedFile.FileName,
		Storage:        types.StorageType(args[2]),
		ChatID:         receivedFile.ChatID,
		MessageID:      receivedFile.MessageID,
		ReplyMessageID: receivedFile.ReplyMessageID,
	})
	query.Edit(fmt.Sprintf("已添加到队列: %s\n当前排队任务数: %d", receivedFile.FileName, queue.Len()))
	return nil
}
