package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/gookit/goutil/maputil"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/model"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

func Start(ctx context.Context, bot *telego.Bot, message telego.Message) {
	if err := dao.CreateUser(message.From.ID); err != nil {
		logger.L.Errorf("Failed to create user: %s", err)
		return
	}
	Help(ctx, bot, message)
}

func Help(ctx context.Context, bot *telego.Bot, message telego.Message) {
	helpText := `
SaveAny Bot - 转存你的 Telegram 文件
命令:
/start - 开始使用
/help - 显示帮助
/silent - 静默模式
/storage - 设置默认存储位置
/save - 保存文件
/clean - 清除文件记录

静默模式: 开启后 Bot 直接保存到收到的文件到默认位置, 不再询问
	`
	ReplyMessage(message, helpText)
}

func ChangeSilentMode(ctx context.Context, bot *telego.Bot, message telego.Message) {
	user, err := dao.GetUserByUserID(message.From.ID)
	if err != nil {
		logger.L.Error(err)
		return
	}
	user.Silent = !user.Silent
	err = dao.UpdateUser(user)
	if err != nil {
		logger.L.Error(err)
		return
	}
	ReplyMessage(message, fmt.Sprintf("已%s静默模式", map[bool]string{true: "开启", false: "关闭"}[user.Silent]))
}

func SetDefaultStorage(ctx context.Context, bot *telego.Bot, message telego.Message) {
	if len(storage.Storages) == 0 {
		ReplyMessage(message, "当前无可用存储端, 请检查配置.")
		return
	}
	_, _, args := telegoutil.ParseCommand(message.Text)
	availableStorages := maputil.Keys(storage.Storages)
	if len(args) == 0 {
		text := EscapeMarkdown("请提供存储位置名称, 可用项:")
		for _, name := range availableStorages {
			text += fmt.Sprintf("\n`%s`", name)
		}
		text += fmt.Sprintf("\n`all`")
		bot.SendMessage(telegoutil.Message(message.Chat.ChatID(), text).WithParseMode(telego.ModeMarkdownV2))
		return
	}
	storageName := args[0]
	if !slice.Contain(availableStorages, storageName) {
		ReplyMessage(message, "参数错误")
		return
	}
	user, err := dao.GetUserByUserID(message.From.ID)
	if err != nil {
		logger.L.Error(err)
		return
	}
	user.DefaultStorage = storageName
	err = dao.UpdateUser(user)
	if err != nil {
		logger.L.Error(err)
		return
	}
	ReplyMessage(message, fmt.Sprintf("已设置默认存储位置为: %s", storageName))
}

func SaveFile(ctx context.Context, bot *telego.Bot, message telego.Message) {
	targetMessage := message.ReplyToMessage
	if targetMessage == nil {
		ReplyMessage(message, "请回复要保存的文件")
		return
	}
	if targetMessage.Document == nil && targetMessage.Video == nil && targetMessage.Audio == nil {
		ReplyMessage(message, "回复的消息不包含文件")
		return
	}
	ctx = context.WithValue(ctx, "force", true)
	HandleFileMessage(ctx, bot, *targetMessage)
}

func CleanReceivedFile(ctx context.Context, bot *telego.Bot, message telego.Message) {
	targetMessage := message.ReplyToMessage
	if targetMessage == nil {
		ReplyMessage(message, "请回复要清除记录的文件")
		return
	}
	if targetMessage.Document == nil && targetMessage.Video == nil && targetMessage.Audio == nil {
		ReplyMessage(message, "回复的消息不包含文件")
		return
	}
	var fileUniqueID string
	if targetMessage.Document != nil {
		fileUniqueID = targetMessage.Document.FileUniqueID
	} else if targetMessage.Video != nil {
		fileUniqueID = targetMessage.Video.FileUniqueID
	} else if targetMessage.Audio != nil {
		fileUniqueID = targetMessage.Audio.FileUniqueID
	}

	if fileUniqueID == "" {
		ReplyMessage(message, "文件信息获取失败")
		return
	}

	if err := dao.DeleteReceivedFileByFileUniqueID(fileUniqueID); err != nil {
		logger.L.Error(err)
		ReplyMessage(message, "删除记录失败")
		return
	}
	ReplyMessage(message, "记录已删除")
}

func HandleFileMessage(ctx context.Context, bot *telego.Bot, message telego.Message) {
	var fileID, fileName string
	if message.Document != nil {
		fileID = message.Document.FileID
		fileName = message.Document.FileName
	} else if message.Video != nil {
		fileID = message.Video.FileID
		fileName = message.Video.FileName
	} else if message.Audio != nil {
		fileID = message.Audio.FileID
		fileName = message.Audio.FileName
	}

	if fileID == "" || fileName == "" {
		ReplyMessage(message, "文件信息获取失败")
		return
	}
	user, err := dao.GetUserByUserID(message.From.ID)
	if err != nil {
		logger.L.Error(err)
		return
	}
	msg, err := ReplyMessage(message, "正在获取文件信息")
	if err != nil {
		logger.L.Error(err)
		return
	}
	file, err := bot.GetFile(&telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.L.Error(err)
		ReplyMessage(message, "获取文件信息失败")
		return
	}
	if ctx.Value("force") == nil {
		receivedFile, _ := dao.GetReceivedFileByFileID(file.FileID)
		if receivedFile != nil && receivedFile.Processing {
			bot.EditMessageText(&telego.EditMessageTextParams{
				ChatID:    message.Chat.ChatID(),
				MessageID: msg.MessageID,
				Text:      "该文件或许正在处理中, 使用 /save 命令回复此文件以强制加入队列\n使用 /clean 命令回复此文件以清除对应的记录",
			})
			return
		}
	}

	err = dao.AddReceivedFile(&model.ReceivedFile{
		FileUniqueID:   file.FileUniqueID,
		FileID:         file.FileID,
		Processing:     false,
		FileName:       fileName,
		FilePath:       file.FilePath,
		FileSize:       file.FileSize,
		MediaGroupID:   message.MediaGroupID,
		ChatID:         message.Chat.ChatID().ID,
		MessageID:      message.MessageID,
		ReplyMessageID: msg.MessageID,
	})

	if err != nil {
		logger.L.Error(err)
		ReplyMessage(message, "创建任务失败")
		return
	}

	if !user.Silent {
		bot.EditMessageText(&telego.EditMessageTextParams{
			ChatID:      message.Chat.ChatID(),
			MessageID:   msg.MessageID,
			Text:        "选择要保存的位置",
			ReplyMarkup: AddTaskReplyMarkup(message.MessageID),
		})
		return
	}

	if user.DefaultStorage == "" {
		bot.EditMessageText(&telego.EditMessageTextParams{
			ChatID:    message.Chat.ChatID(),
			MessageID: msg.MessageID,
			Text:      "请先使用 /storage 命令设置默认存储位置, 或者关闭静默模式",
		})
		return
	}

	queue.AddTask(types.Task{
		Ctx:            context.TODO(),
		FileID:         file.FileID,
		Status:         types.Pending,
		FileName:       fileName,
		FilePath:       file.FilePath,
		Storage:        types.StorageType(user.DefaultStorage),
		ChatID:         message.Chat.ChatID().ID,
		ReplyMessageID: msg.MessageID,
	})
}

func AddToQueue(ctx context.Context, bot *telego.Bot, query telego.CallbackQuery) {
	args := strings.Split(query.Data, " ")
	messageID, _ := strconv.Atoi(args[1])
	receivedFile, err := dao.GetReceivedFileByChatAndMessageID(query.Message.GetChat().ID, messageID)
	if err != nil {
		logger.L.Error(err)
		bot.AnswerCallbackQuery(telegoutil.CallbackQuery(query.ID).WithShowAlert().WithText("获取文件信息失败").WithCacheTime(5))
		return
	}
	queue.AddTask(types.Task{
		Ctx:            context.TODO(),
		FileID:         receivedFile.FileID,
		Status:         types.Pending,
		FileName:       receivedFile.FileName,
		FilePath:       receivedFile.FilePath,
		Storage:        types.StorageType(args[2]),
		ChatID:         receivedFile.ChatID,
		ReplyMessageID: receivedFile.ReplyMessageID,
	})
	bot.EditMessageText(&telego.EditMessageTextParams{
		Text:      fmt.Sprintf("已添加到队列, 当前排队中的任务数: %d", queue.Len()),
		MessageID: query.Message.GetMessageID(),
		ChatID:    telegoutil.ID(receivedFile.ChatID),
	})
}
