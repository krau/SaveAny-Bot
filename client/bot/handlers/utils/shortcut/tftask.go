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
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/batchtfile"
	tftask "github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

// 创建一个 tfile.TGFileTask 并添加到任务队列中, 以编辑消息的方式反馈结果
func CreateAndAddTGFileTaskWithEdit(ctx *ext.Context, userID int64, stor storage.Storage, dirPath string, file tfile.TGFileMessage, trackMsgID int, conflictStrategy ...string) error {
	logger := log.FromContext(ctx)
	strategy := firstConflictStrategy(conflictStrategy)
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		logger.Errorf("Failed to get user by chat ID: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID: trackMsgID,
			Message: i18n.T(i18nk.BotMsgCommonErrorGetUserWithErrFailed, map[string]any{
				"Error": err.Error(),
			}),
		})
		return dispatcher.EndGroups
	}
	if strategy == "" {
		strategy = userConflictStrategy(user)
	}
	if user.ApplyRule && user.Rules != nil {
		matched, matchedStorageName, matchedDirPath := ruleutil.ApplyRule(ctx, user.Rules, ruleutil.NewInput(file))
		if !matched {
			goto startCreateTask
		}
		if matchedDirPath != "" {
			dirPath = matchedDirPath.String()
		}
		if matchedStorageName.Usable() {
			stor, err = storage.GetStorageByUserIDAndName(ctx, user.ChatID, matchedStorageName.String())
			if err != nil {
				logger.Errorf("Failed to get storage by user ID and name: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID: trackMsgID,
					Message: i18n.T(i18nk.BotMsgCommonErrorGetStorageFailed, map[string]any{
						"Error": err.Error(),
					}),
				})
				return dispatcher.EndGroups
			}
		}
	}
startCreateTask:
	storagePath := path.Join(dirPath, file.Name())
	if strategy == tcbdata.ConflictStrategyAsk && stor.Exists(ctx, storagePath) {
		return promptTGFileConflictStrategy(ctx, userID, stor.Name(), dirPath, []tfile.TGFileMessage{file}, false, []string{fmt.Sprintf("[%s]:%s", stor.Name(), storagePath)}, trackMsgID)
	}
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	if strategy == tcbdata.ConflictStrategyOverwrite {
		injectCtx = storage.WithOverwrite(injectCtx)
	}
	if strategy == tcbdata.ConflictStrategySkip && stor.Exists(ctx, storagePath) {
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID: trackMsgID,
			Message: i18n.T(i18nk.BotMsgCommonInfoAllConflictFilesSkipped, map[string]any{
				"Skipped": file.Name(),
			}),
			ReplyMarkup: nil,
		})
		return dispatcher.EndGroups
	}
	taskid := xid.New().String()
	task, err := tftask.NewTGFileTask(taskid, injectCtx, file, stor, storagePath,
		tftask.NewProgressTrack(
			trackMsgID,
			userID))
	if err != nil {
		logger.Errorf("create task failed: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID: trackMsgID,
			Message: i18n.T(i18nk.BotMsgCommonErrorTaskCreateFailed, map[string]any{
				"Error": err.Error(),
			}),
		})
		return dispatcher.EndGroups
	}
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("add task failed: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID: trackMsgID,
			Message: i18n.T(i18nk.BotMsgCommonErrorTaskAddFailed, map[string]any{
				"Error": err.Error(),
			}),
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
func CreateAndAddBatchTGFileTaskWithEdit(ctx *ext.Context, userID int64, stor storage.Storage, dirPath string, files []tfile.TGFileMessage, trackMsgID int, conflictStrategy ...string) error {
	logger := log.FromContext(ctx)
	strategy := firstConflictStrategy(conflictStrategy)
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		logger.Errorf("Failed to get user by chat ID: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID: trackMsgID,
			Message: i18n.T(i18nk.BotMsgCommonErrorGetUserWithErrFailed, map[string]any{
				"Error": err.Error(),
			}),
		})
		return dispatcher.EndGroups
	}
	if strategy == "" {
		strategy = userConflictStrategy(user)
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
		if !storName.Usable() {
			storname = stor.Name()
		}
		return storname, dirP
	}

	skipped := make([]string, 0)
	conflicts := make([]string, 0)
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
					ID: trackMsgID,
					Message: i18n.T(i18nk.BotMsgCommonErrorGetStorageFailed, map[string]any{
						"Error": err.Error(),
					}),
				})
				return dispatcher.EndGroups
			}
		}
		if !dirPath.NeedNewForAlbum() {
			storPath := path.Join(dirPath.String(), file.Name())
			if fileStor.Exists(ctx, storPath) {
				if strategy == tcbdata.ConflictStrategyAsk {
					conflicts = append(conflicts, fmt.Sprintf("[%s]:%s", fileStor.Name(), storPath))
				}
			}
			if strategy == tcbdata.ConflictStrategySkip && fileStor.Exists(ctx, storPath) {
				skipped = append(skipped, file.Name())
				continue
			}
			elem, err := batchtfile.NewTaskElement(fileStor, storPath, file)
			if err != nil {
				logger.Errorf("Failed to create task element: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID: trackMsgID,
					Message: i18n.T(i18nk.BotMsgCommonErrorTaskCreateFailed, map[string]any{
						"Error": err.Error(),
					}),
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
			afstorPath := path.Join(dirPath, albumDir, af.file.Name())
			if albumStor.Exists(ctx, afstorPath) {
				if strategy == tcbdata.ConflictStrategyAsk {
					conflicts = append(conflicts, fmt.Sprintf("[%s]:%s", albumStor.Name(), afstorPath))
				}
			}
			if strategy == tcbdata.ConflictStrategySkip && albumStor.Exists(ctx, afstorPath) {
				skipped = append(skipped, af.file.Name())
				continue
			}
			elem, err := batchtfile.NewTaskElement(albumStor, afstorPath, af.file)
			if err != nil {
				logger.Errorf("Failed to create task element for album file: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID: trackMsgID,
					Message: i18n.T(i18nk.BotMsgCommonErrorTaskCreateFailed, map[string]any{
						"Error": err.Error(),
					}),
				})
				return dispatcher.EndGroups
			}
			elems = append(elems, *elem)
		}
	}

	if strategy == tcbdata.ConflictStrategyAsk && len(conflicts) > 0 {
		return promptTGFileConflictStrategy(ctx, userID, stor.Name(), dirPath, files, true, conflicts, trackMsgID)
	}

	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	if strategy == tcbdata.ConflictStrategyOverwrite {
		injectCtx = storage.WithOverwrite(injectCtx)
	}
	if len(elems) == 0 {
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID: trackMsgID,
			Message: i18n.T(i18nk.BotMsgCommonInfoAllConflictFilesSkipped, map[string]any{
				"Skipped": strings.Join(skipped, "\n"),
			}),
			ReplyMarkup: nil,
		})
		return dispatcher.EndGroups
	}
	taskid := xid.New().String()
	task := batchtfile.NewBatchTGFileTask(taskid, injectCtx, elems, batchtfile.NewProgressTrackerWithSkipped(trackMsgID, userID, skipped), true)
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("Failed to add batch task: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID: trackMsgID,
			Message: i18n.T(i18nk.BotMsgCommonErrorTaskAddFailed, map[string]any{
				"Error": err.Error(),
			}),
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:          trackMsgID,
		Message:     buildBatchAddedMessage(len(elems), skipped),
		ReplyMarkup: nil,
	})
	return dispatcher.EndGroups
}

func promptTGFileConflictStrategy(ctx *ext.Context, userID int64, storageName, dirPath string, files []tfile.TGFileMessage, asBatch bool, conflicts []string, trackMsgID int) error {
	markup, err := msgelem.BuildConflictStrategyMarkup(tcbdata.Add{
		TaskType:         tasktype.TaskTypeTgfiles,
		SelectedStorName: storageName,
		SettedDir:        true,
		SelectedDirPath:  dirPath,
		Files:            files,
		AsBatch:          asBatch,
	})
	if err != nil {
		return err
	}
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:          trackMsgID,
		Message:     i18n.T(i18nk.BotMsgCommonPromptSelectConflictStrategy, map[string]any{"Files": formatConflictPaths(conflicts)}),
		ReplyMarkup: markup,
	})
	return dispatcher.EndGroups
}

func formatConflictPaths(conflicts []string) string {
	const maxConflictLines = 10
	if len(conflicts) <= maxConflictLines {
		return strings.Join(conflicts, "\n")
	}
	return strings.Join(conflicts[:maxConflictLines], "\n") + "\n" + i18n.T(i18nk.BotMsgCommonPromptConflictMoreFiles, map[string]any{
		"Count": len(conflicts) - maxConflictLines,
	})
}

func firstConflictStrategy(strategies []string) string {
	if len(strategies) == 0 {
		return ""
	}
	return strategies[0]
}

func userConflictStrategy(user *database.User) string {
	if user != nil && tcbdata.IsConflictStrategy(user.ConflictStrategy) {
		return user.ConflictStrategy
	}
	return tcbdata.ConflictStrategyRename
}

func buildBatchAddedMessage(count int, skipped []string) string {
	if len(skipped) == 0 {
		return i18n.T(i18nk.BotMsgCommonInfoBatchTasksAdded, map[string]any{
			"Count": count,
		})
	}
	return i18n.T(i18nk.BotMsgCommonInfoBatchTasksAddedWithSkipped, map[string]any{
		"Count":   count,
		"Skipped": strings.Join(skipped, "\n"),
	})
}
