package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/parsed"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

func handleAddCallback(ctx *ext.Context, update *ext.Update) error {
	dataid := strings.Split(string(update.CallbackQuery.Data), " ")[1]
	data, err := shortcut.GetCallbackDataWithAnswer[tcbdata.Add](ctx, update, dataid)
	if err != nil {
		return err
	}
	queryID := update.CallbackQuery.GetQueryID()
	msgID := update.CallbackQuery.GetMsgID()
	userID := update.CallbackQuery.GetUserID()

	selectedStorage, err := storage.GetStorageByUserIDAndName(ctx, userID, data.SelectedStorName)
	if err != nil {
		log.FromContext(ctx).Errorf("Failed to get storage: %s", err)
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "存储获取失败: "+err.Error()))
		return dispatcher.EndGroups
	}
	dirs, err := database.GetDirsByUserChatIDAndStorageName(ctx, userID, data.SelectedStorName)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("获取用户目录失败: %w", err)
	}

	if !data.SettedDir && len(dirs) != 0 {
		// ask for directory selection
		markup, err := msgelem.BuildSetDirKeyboard(dirs, dataid)
		if err != nil {
			log.FromContext(ctx).Errorf("Failed to build directory keyboard: %s", err)
			ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "目录键盘构建失败: "+err.Error()))
			return dispatcher.EndGroups
		}
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:          update.CallbackQuery.GetMsgID(),
			Message:     "请选择要存储到的目录",
			ReplyMarkup: markup,
		})
		return dispatcher.EndGroups
	}

	dirPath := ""
	if data.DirID != 0 {
		dir, err := database.GetDirByID(ctx, data.DirID)
		if err != nil {
			ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, "获取目录失败: "+err.Error()))
			return dispatcher.EndGroups
		}
		dirPath = dir.Path
	}

	switch data.TaskType {
	case tasktype.TaskTypeTgfiles:
		if data.AsBatch {
			return shortcut.CreateAndAddBatchTGFileTaskWithEdit(ctx, userID, selectedStorage, dirPath, data.Files, msgID)
		}
		return shortcut.CreateAndAddTGFileTaskWithEdit(ctx, userID, selectedStorage, dirPath, data.Files[0], msgID)
	case tasktype.TaskTypeTphpics:
		return shortcut.CreateAndAddtelegraphWithEdit(ctx, userID, data.TphPageNode, data.TphDirPath, data.TphPics, selectedStorage, msgID)
	case tasktype.TaskTypeParseditem:
		task := parsed.NewTask(xid.New().String(), ctx, selectedStorage, selectedStorage.JoinStoragePath(dirPath), data.ParsedItem)
		if err := core.AddTask(ctx, task); err != nil {
			log.FromContext(ctx).Errorf("Failed to add task: %s", err)
			ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
				ID:      msgID,
				Message: "任务添加失败: " + err.Error(),
			})
			return dispatcher.EndGroups
		}
		text, entities := msgelem.BuildTaskAddedEntities(ctx, data.ParsedItem.Title, core.GetLength(ctx))
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:       msgID,
			Message:  text,
			Entities: entities,
		})
	default:
		log.FromContext(ctx).Errorf("Unsupported task type: %s", data.TaskType)
	}
	return dispatcher.EndGroups
}
