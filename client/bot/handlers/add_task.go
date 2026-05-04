package handlers

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/ruleutil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
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
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, i18n.T(i18nk.BotMsgCommonErrorGetStorageFailed, map[string]any{
			"Error": err.Error(),
		})))
		return dispatcher.EndGroups
	}
	dirs, err := database.GetDirsByUserChatIDAndStorageName(ctx, userID, data.SelectedStorName)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get user directories: %w", err)
	}

	if !data.SettedDir && len(dirs) != 0 {
		// ask for directory selection
		markup, err := msgelem.BuildSetDirMarkupForAdd(dirs, dataid)
		if err != nil {
			log.FromContext(ctx).Errorf("Failed to build directory keyboard: %s", err)
			ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, i18n.T(i18nk.BotMsgCommonErrorBuildStorageSelectKeyboardFailed, map[string]any{
				"Error": err.Error(),
			})))
			return dispatcher.EndGroups
		}
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:          update.CallbackQuery.GetMsgID(),
			Message:     i18n.T(i18nk.BotMsgCommonPromptSelectDir, nil),
			ReplyMarkup: markup,
		})
		return dispatcher.EndGroups
	}

	dirPath := ""
	if data.DirID != 0 {
		dir, err := database.GetDirByID(ctx, data.DirID)
		if err != nil {
			ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, i18n.T(i18nk.BotMsgCommonErrorGetDirFailed, map[string]any{
				"Error": err.Error(),
			})))
			return dispatcher.EndGroups
		}
		dirPath = dir.Path
	} else if data.SelectedDirPath != "" {
		dirPath = data.SelectedDirPath
	}

	switch data.TaskType {
	case tasktype.TaskTypeTgfiles:
		conflicts, err := findTGFileConflicts(ctx, userID, selectedStorage, dirPath, data.Files)
		if err != nil {
			ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, err.Error()))
			return dispatcher.EndGroups
		}
		if len(conflicts) > 0 && data.ConflictStrategy == "" {
			markup, err := msgelem.BuildConflictStrategyMarkup(data)
			if err != nil {
				ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, i18n.T(i18nk.BotMsgCommonErrorBuildStorageSelectKeyboardFailed, map[string]any{
					"Error": err.Error(),
				})))
				return dispatcher.EndGroups
			}
			ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
				ID:          update.CallbackQuery.GetMsgID(),
				Message:     i18n.T(i18nk.BotMsgCommonPromptSelectConflictStrategy, map[string]any{"Files": formatConflictFiles(conflicts)}),
				ReplyMarkup: markup,
			})
			return dispatcher.EndGroups
		}
		if data.AsBatch {
			return shortcut.CreateAndAddBatchTGFileTaskWithEdit(ctx, userID, selectedStorage, dirPath, data.Files, msgID, data.ConflictStrategy)
		}
		return shortcut.CreateAndAddTGFileTaskWithEdit(ctx, userID, selectedStorage, dirPath, data.Files[0], msgID, data.ConflictStrategy)
	case tasktype.TaskTypeTphpics:
		return shortcut.CreateAndAddtelegraphWithEdit(ctx, userID, data.TphPageNode, data.TphDirPath, data.TphPics, selectedStorage, msgID)
	case tasktype.TaskTypeParseditem:
		if len(data.ParsedItem.Resources) > 1 {
			dirPath = path.Join(dirPath, fsutil.NormalizePathname(data.ParsedItem.Title))
		}
		shortcut.CreateAndAddParsedTaskWithEdit(ctx, selectedStorage, dirPath, data.ParsedItem, msgID, userID)
	case tasktype.TaskTypeDirectlinks:
		shortcut.CreateAndAddDirectTaskWithEdit(ctx, selectedStorage, dirPath, data.DirectLinks, msgID, userID)
	case tasktype.TaskTypeAria2:
		client := GetAria2Client()
		if client == nil {
			ctx.AnswerCallback(msgelem.AlertCallbackAnswer(queryID, i18n.T(i18nk.BotMsgAria2ErrorAria2ClientInitFailed, map[string]any{
				"Error": "aria2 client not initialized",
			})))
			return dispatcher.EndGroups
		}
		shortcut.CreateAndAddAria2TaskWithEdit(ctx, selectedStorage, dirPath, data.Aria2URIs, client, msgID, userID)
	case tasktype.TaskTypeYtdlp:
		shortcut.CreateAndAddYtdlpTaskWithEdit(ctx, selectedStorage, dirPath, data.YtdlpURLs, data.YtdlpFlags, msgID, userID)
	case tasktype.TaskTypeTransfer:
		return handleTransferCallback(ctx, userID, selectedStorage, dirPath, data, msgID)
	default:
		return fmt.Errorf("unexcept task type: %s", data.TaskType)
	}
	return dispatcher.EndGroups
}

type tgFileConflict struct {
	Name        string
	StorageName string
	Path        string
}

func findTGFileConflicts(ctx *ext.Context, userID int64, stor storage.Storage, dirPath string, files []tfile.TGFileMessage) ([]tgFileConflict, error) {
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return nil, errors.New(i18n.T(i18nk.BotMsgCommonErrorGetUserWithErrFailed, map[string]any{"Error": err.Error()}))
	}
	useRule := user.ApplyRule && user.Rules != nil
	conflicts := make([]tgFileConflict, 0)

	resolve := func(file tfile.TGFileMessage) (storage.Storage, string, error) {
		fileStor := stor
		fileDirPath := dirPath
		if useRule {
			matched, matchedStorageName, matchedDirPath := ruleutil.ApplyRule(ctx, user.Rules, ruleutil.NewInput(file))
			if matched {
				if matchedDirPath != "" {
					fileDirPath = matchedDirPath.String()
				}
				if matchedStorageName.Usable() {
					var err error
					fileStor, err = storage.GetStorageByUserIDAndName(ctx, user.ChatID, matchedStorageName.String())
					if err != nil {
						return nil, "", errors.New(i18n.T(i18nk.BotMsgCommonErrorGetStorageFailed, map[string]any{"Error": err.Error()}))
					}
				}
			}
		}
		return fileStor, fileDirPath, nil
	}

	albumFiles := make(map[int64][]tfile.TGFileMessage)
	for _, file := range files {
		fileStor, fileDirPath, err := resolve(file)
		if err != nil {
			return nil, err
		}
		if ruleutil.MatchedDirPath(fileDirPath).NeedNewForAlbum() {
			groupID, isGroup := file.Message().GetGroupedID()
			if isGroup && groupID != 0 {
				albumFiles[groupID] = append(albumFiles[groupID], file)
			}
			continue
		}
		storagePath := path.Join(fileDirPath, file.Name())
		if fileStor.Exists(ctx, storagePath) {
			conflicts = append(conflicts, tgFileConflict{Name: file.Name(), StorageName: fileStor.Name(), Path: storagePath})
		}
	}

	for _, afiles := range albumFiles {
		if len(afiles) <= 1 {
			continue
		}
		albumDir := strings.TrimSuffix(path.Base(afiles[0].Name()), path.Ext(afiles[0].Name()))
		albumStor, _, err := resolve(afiles[0])
		if err != nil {
			return nil, err
		}
		for _, file := range afiles {
			storagePath := path.Join(dirPath, albumDir, file.Name())
			if albumStor.Exists(ctx, storagePath) {
				conflicts = append(conflicts, tgFileConflict{Name: file.Name(), StorageName: albumStor.Name(), Path: storagePath})
			}
		}
	}
	return conflicts, nil
}

func formatConflictFiles(conflicts []tgFileConflict) string {
	const maxConflictLines = 10
	var b strings.Builder
	for i, conflict := range conflicts {
		if i >= maxConflictLines {
			fmt.Fprint(&b, i18n.T(i18nk.BotMsgCommonPromptConflictMoreFiles, map[string]any{
				"Count": len(conflicts) - maxConflictLines,
			}))
			break
		}
		fmt.Fprintf(&b, "- [%s]:%s\n", conflict.StorageName, conflict.Path)
	}
	return strings.TrimSpace(b.String())
}
