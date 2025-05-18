package bot

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/types"
	"gorm.io/gorm"
)

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
	addToDir := args[0] == "add_to_dir" // 已经选择了路径
	cbDataId, _ := strconv.Atoi(args[1])
	cbData, err := dao.GetCallbackData(uint(cbDataId))
	if err != nil {
		common.Log.Errorf("获取回调数据失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "获取回调数据失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}

	data := strings.Split(cbData, " ")
	fileChatID, _ := strconv.Atoi(data[0])
	fileMessageID, _ := strconv.Atoi(data[1])
	storageName := data[2]
	dirIdInt, _ := strconv.Atoi(data[3])
	dirId := uint(dirIdInt)

	user, err := dao.GetUserByChatID(update.CallbackQuery.UserID)
	if err != nil {
		common.Log.Errorf("获取用户失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.QueryID,
			Alert:     true,
			Message:   "获取用户失败",
			CacheTime: 5,
		})
		return dispatcher.EndGroups
	}

	if !addToDir {
		dirs, err := dao.GetDirsByUserIDAndStorageName(user.ID, storageName)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			common.Log.Errorf("获取路径失败: %s", err)
			ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
				QueryID:   update.CallbackQuery.QueryID,
				Alert:     true,
				Message:   "获取路径失败",
				CacheTime: 5,
			})
			return dispatcher.EndGroups
		}
		if len(dirs) != 0 {
			markup, err := getSelectDirMarkup(fileChatID, fileMessageID, storageName, dirs)
			if err != nil {
				common.Log.Errorf("获取路径失败: %s", err)
				ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
					QueryID:   update.CallbackQuery.QueryID,
					Alert:     true,
					Message:   "获取路径失败",
					CacheTime: 5,
				})
				return dispatcher.EndGroups
			}
			_, err = ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
				ID:          update.CallbackQuery.GetMsgID(),
				Message:     "请选择要保存到的路径",
				ReplyMarkup: markup,
			})
			if err != nil {
				common.Log.Errorf("编辑消息失败: %s", err)
			}
			return dispatcher.EndGroups
		}
	}

	common.Log.Tracef("Got add to queue: chatID: %d, messageID: %d, storage: %s", fileChatID, fileMessageID, storageName)
	record, err := dao.GetReceivedFileByChatAndMessageID(int64(fileChatID), fileMessageID)
	if err != nil {
		common.Log.Errorf("获取记录失败: %s", err)
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
			common.Log.Errorf("更新记录失败: %s", err)
		}
	}

	var dir *dao.Dir
	if addToDir && dirId != 0 {
		dir, err = dao.GetDirByID(dirId)
		if err != nil {
			common.Log.Errorf("获取路径失败: %s", err)
			ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
				QueryID:   update.CallbackQuery.QueryID,
				Alert:     true,
				Message:   "获取路径失败",
				CacheTime: 5,
			})
			return dispatcher.EndGroups
		}
	}

	var task types.Task
	if record.IsTelegraph {
		task = types.Task{
			Ctx:            ctx,
			Status:         types.Pending,
			IsTelegraph:    true,
			TelegraphURL:   record.TelegraphURL,
			StorageName:    storageName,
			FileChatID:     record.ChatID,
			FileMessageID:  record.MessageID,
			ReplyMessageID: record.ReplyMessageID,
			ReplyChatID:    record.ReplyChatID,
			UserID:         update.GetUserChat().GetID(),
		}
		if dir != nil {
			task.StoragePath = path.Join(dir.Path, record.FileName)
		}
	} else {
		file, err := FileFromMessage(ctx, record.ChatID, record.MessageID, record.FileName)
		if err != nil {
			common.Log.Errorf("获取消息中的文件失败: %s", err)
			ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
				QueryID:   update.CallbackQuery.QueryID,
				Alert:     true,
				Message:   fmt.Sprintf("获取消息中的文件失败: %s", err),
				CacheTime: 5,
			})
			return dispatcher.EndGroups
		}

		task = types.Task{
			Ctx:            ctx,
			Status:         types.Pending,
			FileDBID:       record.ID,
			File:           file,
			StorageName:    storageName,
			FileChatID:     record.ChatID,
			ReplyMessageID: record.ReplyMessageID,
			FileMessageID:  record.MessageID,
			ReplyChatID:    record.ReplyChatID,
			UserID:         update.GetUserChat().GetID(),
		}
		if dir != nil {
			task.StoragePath = path.Join(dir.Path, file.FileName)
		}
	}

	queue.AddTask(&task)

	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	text := fmt.Sprintf("已添加到任务队列\n文件名: %s\n当前排队任务数: %d", record.FileName, queue.Len())
	if err := styling.Perform(&entityBuilder,
		styling.Plain("已添加到任务队列\n文件名: "),
		styling.Code(record.FileName),
		styling.Plain("\n当前排队任务数: "),
		styling.Bold(strconv.Itoa(queue.Len())),
	); err != nil {
		common.Log.Errorf("Failed to build entity: %s", err)
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
