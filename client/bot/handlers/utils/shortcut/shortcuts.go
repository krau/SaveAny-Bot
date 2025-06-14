// Some shortcuts for duplicate code in handlers, they should return dispatcher errors
package shortcut

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/types"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/batchtftask"
	"github.com/krau/SaveAny-Bot/core/tftask"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

// 获取消息中的文件并回复等待消息, 返回等待消息, 获取到的文件
func GetFileFromMessageWithReply(ctx *ext.Context, update *ext.Update, message tg.Message, tfileopts ...tfile.FromMediaOptions) (replied *types.Message,
	file tfile.TGFile, err error,
) {
	logger := log.FromContext(ctx)
	media := message.Media
	supported := mediautil.IsSupported(media)
	if !supported {
		ctx.Reply(update, ext.ReplyTextString("不支持的消息类型"), nil)
		return nil, nil, dispatcher.EndGroups
	}

	replied, err = ctx.Reply(update, ext.ReplyTextString("正在获取文件信息..."), nil)
	if err != nil {
		logger.Errorf("Failed to reply: %s", err)
		return nil, nil, dispatcher.EndGroups
	}
	options := make([]tfile.FromMediaOptions, 0, len(tfileopts)+1)
	if len(tfileopts) > 0 {
		options = tfileopts
	} else {
		options = append(options, tfile.WithNameIfEmpty(tgutil.GenFileNameFromMessage(message)))
	}
	file, err = tfile.FromMedia(media, options...)
	if err != nil {
		logger.Errorf("Failed to get file from media: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取文件失败: "+err.Error()), nil)
		return nil, nil, dispatcher.EndGroups
	}
	return replied, file, nil
}

// 创建一个 tftask.TGFileTask 并添加到任务队列中, 以编辑消息的方式反馈结果
func CreateAndAddTGFileTaskWithEdit(ctx *ext.Context, stor storage.Storage, file tfile.TGFile, chatID int64, trackMsgID int) error {
	logger := log.FromContext(ctx)
	storagePath := stor.JoinStoragePath(file.Name())
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	taskid := xid.New().String()
	task, err := tftask.NewTGFileTask(taskid, injectCtx, file, ctx.Raw, stor, storagePath,
		tftask.NewProgressTrack(
			trackMsgID,
			chatID))
	if err != nil {
		logger.Errorf("create task failed: %s", err)
		ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "创建任务失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("add task failed: %s", err)
		ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "添加任务失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	text, entities := msgelem.BuildTaskAddedEntities(ctx, file.Name(), core.GetLength(injectCtx))
	ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
		ID:       trackMsgID,
		Message:  text,
		Entities: entities,
	})

	return dispatcher.EndGroups
}

type EditMessageFunc func(text string, markup tg.ReplyMarkupClass)

// 获取链接中的文件并回复等待消息
func GetFilesFromUpdateLinkMessageWithReplyEdit(ctx *ext.Context, update *ext.Update) (replied *types.Message, files []tfile.TGFile, editReplied EditMessageFunc, err error) {
	logger := log.FromContext(ctx)
	msgLinks := re.TgMessageLinkRegexp.FindAllString(update.EffectiveMessage.GetMessage(), -1)
	if len(msgLinks) == 0 {
		logger.Warn("no matched message links but called handleMessageLink")
		return nil, nil, nil, dispatcher.EndGroups
	}
	replied, err = ctx.Reply(update, ext.ReplyTextString("正在获取消息..."), nil)
	if err != nil {
		logger.Errorf("failed to reply: %s", err)
		return nil, nil, nil, dispatcher.EndGroups
	}
	editReplied = func(text string, markup tg.ReplyMarkupClass) {
		if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:          replied.ID,
			Message:     text,
			ReplyMarkup: markup,
		}); err != nil {
			logger.Errorf("failed to edit message: %s", err)
		}
	}

	files = make([]tfile.TGFile, 0, len(msgLinks))
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
		return nil, nil, nil, dispatcher.EndGroups
	}
	return replied, files, editReplied, nil
}

func GetCallbackDataWithAnswer[DataType any](ctx *ext.Context, update *ext.Update, dataid string) (DataType, error) {
	data, ok := cache.Get[DataType](dataid)
	if !ok {
		log.FromContext(ctx).Warnf("Invalid data ID: %s", dataid)
		queryID := update.CallbackQuery.GetQueryID()
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "数据已过期或无效"))
		var zero DataType
		return zero, dispatcher.EndGroups
	}
	return data, nil
}

// 创建一个批量 batchtftask.BatchTGFileTask 并添加到任务队列中, 以编辑消息的方式反馈结果
func CreateAndAddBatchTGFileTaskWithEdit(ctx *ext.Context, stor storage.Storage, files []tfile.TGFile, chatID int64, trackMsgID int) error {
	logger := log.FromContext(ctx)
	elems := make([]batchtftask.TaskElement, 0, len(files))
	for _, file := range files {
		storPath := stor.JoinStoragePath(file.Name())
		elem, err := batchtftask.NewTaskElement(stor, storPath, file)
		if err != nil {
			logger.Errorf("Failed to create task element: %s", err)
			ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
				ID:      trackMsgID,
				Message: "任务创建失败: " + err.Error(),
			})
			return dispatcher.EndGroups
		}
		elems = append(elems, *elem)
	}
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	taskid := xid.New().String()
	task := batchtftask.NewBatchTGFileTask(taskid, injectCtx, elems, ctx.Raw, batchtftask.NewProgressTracker(trackMsgID, chatID), true)
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("Failed to add batch task: %s", err)
		ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "批量任务添加失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
		ID:          trackMsgID,
		Message:     fmt.Sprintf("已添加批量任务, 共 %d 个文件", len(files)), // TODO: stylng message
		ReplyMarkup: nil,
	})
	return dispatcher.EndGroups
}
