package shortcut

import (
	"fmt"
	"path"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/ruleutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/batchtfile"
	tftask "github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

// 创建一个 tfile.TGFileTask 并添加到任务队列中, 以编辑消息的方式反馈结果
func CreateAndAddTGFileTaskWithEdit(ctx *ext.Context, userID int64, stor storage.Storage, dirPath string, file tfile.TGFileMessage, trackMsgID int) error {
	logger := log.FromContext(ctx)
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		logger.Errorf("Failed to get user by chat ID: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "获取用户失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	if user.ApplyRule && user.Rules != nil {
		matched, matchedStorageName, matchedDirPath := ruleutil.ApplyRule(ctx, user.Rules, ruleutil.NewInput(file))
		if !matched {
			goto startCreateTask
		}
		if matchedDirPath != "" {
			dirPath = matchedDirPath.String()
		}
		if matchedStorageName.IsUsable() {
			stor, err = storage.GetStorageByUserIDAndName(ctx, user.ChatID, matchedStorageName.String())
			if err != nil {
				logger.Errorf("Failed to get storage by user ID and name: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID:      trackMsgID,
					Message: "获取存储失败: " + err.Error(),
				})
				return dispatcher.EndGroups
			}
		}
	}
startCreateTask:
	storagePath := stor.JoinStoragePath(path.Join(dirPath, file.Name()))
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	taskid := xid.New().String()
	task, err := tftask.NewTGFileTask(taskid, injectCtx, file, stor, storagePath,
		tftask.NewProgressTrack(
			trackMsgID,
			userID))
	if err != nil {
		logger.Errorf("create task failed: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "创建任务失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("add task failed: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "添加任务失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	text, entities := msgelem.BuildTaskAddedEntities(ctx, file.Name(), core.GetLength(injectCtx))
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:       trackMsgID,
		Message:  text,
		Entities: entities,
	})

	return dispatcher.EndGroups
}

// 创建一个 batchtfile.BatchTGFileTask 并添加到任务队列中, 以编辑消息的方式反馈结果
func CreateAndAddBatchTGFileTaskWithEdit(ctx *ext.Context, userID int64, stor storage.Storage, dirPath string, files []tfile.TGFileMessage, trackMsgID int) error {
	logger := log.FromContext(ctx)
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		logger.Errorf("Failed to get user by chat ID: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "获取用户失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}

	useRule := user.ApplyRule && user.Rules != nil

	applyRule := func(file tfile.TGFileMessage) (string, ruleutil.MatchedDirPath) {
		if !useRule {
			return stor.Name(), ruleutil.MatchedDirPath(dirPath)
		}
		matched, storName, dirP := ruleutil.ApplyRule(ctx, user.Rules, ruleutil.NewInput(file))
		if !matched {
			return stor.Name(), ruleutil.MatchedDirPath(dirPath)
		}
		storname := storName.String()
		if !storName.IsUsable() {
			storname = stor.Name()
		}
		return storname, dirP
	}

	elems := make([]batchtfile.TaskElement, 0, len(files))
	type albumFile struct {
		file    tfile.TGFileMessage
		storage storage.Storage
	}
	albumFiles := make(map[int64][]albumFile, 0)
	for _, file := range files {
		storName, dirPath := applyRule(file)
		fileStor := stor
		if storName != stor.Name() && storName != "" {
			fileStor, err = storage.GetStorageByUserIDAndName(ctx, user.ChatID, storName)
			if err != nil {
				logger.Errorf("Failed to get storage by user ID and name: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID:      trackMsgID,
					Message: "获取存储失败: " + err.Error(),
				})
				return dispatcher.EndGroups
			}
		}
		if !dirPath.NeedNewForAlbum() {
			storPath := fileStor.JoinStoragePath(path.Join(dirPath.String(), file.Name()))
			elem, err := batchtfile.NewTaskElement(fileStor, storPath, file)
			if err != nil {
				logger.Errorf("Failed to create task element: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID:      trackMsgID,
					Message: "任务创建失败: " + err.Error(),
				})
				return dispatcher.EndGroups
			}
			elems = append(elems, *elem)
		} else {
			groupId, isGroup := file.Message().GetGroupedID()
			if !isGroup || groupId == 0 {
				logger.Warnf("File %s is not in a group, skipping album handling", file.Name())
				continue
			}
			if _, ok := albumFiles[groupId]; !ok {
				albumFiles[groupId] = make([]albumFile, 0)
			}
			albumFiles[groupId] = append(albumFiles[groupId], albumFile{
				file:    file,
				storage: fileStor,
			})
		}
	}
	for _, afiles := range albumFiles {
		if len(afiles) <= 1 {
			continue
		}
		// 对于需要新建目录的文件, 将第一个文件的文件名(去除扩展名)作为目录名
		// 存储以第一个文件的存储为准
		albumDir := strings.TrimSuffix(path.Base(afiles[0].file.Name()), path.Ext(afiles[0].file.Name()))
		albumStor := afiles[0].storage
		for _, af := range afiles {
			afstorPath := af.storage.JoinStoragePath(path.Join(dirPath, albumDir, af.file.Name()))
			elem, err := batchtfile.NewTaskElement(albumStor, afstorPath, af.file)
			if err != nil {
				logger.Errorf("Failed to create task element for album file: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID:      trackMsgID,
					Message: "任务创建失败: " + err.Error(),
				})
				return dispatcher.EndGroups
			}
			elems = append(elems, *elem)
		}
	}

	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	taskid := xid.New().String()
	task := batchtfile.NewBatchTGFileTask(taskid, injectCtx, elems, batchtfile.NewProgressTracker(trackMsgID, userID), true)
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("Failed to add batch task: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "批量任务添加失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:          trackMsgID,
		Message:     fmt.Sprintf("已添加批量任务, 共 %d 个文件", len(files)),
		ReplyMarkup: nil,
	})
	return dispatcher.EndGroups
}
