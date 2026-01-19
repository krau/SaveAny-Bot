package handlers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/transfer"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func handleTransferCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strutil.ParseArgsRespectQuotes(update.EffectiveMessage.Text)

	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTransferUsage, nil)), nil)
		return dispatcher.EndGroups
	}

	// Parse source: storage_name:/path
	sourceParts := strings.SplitN(args[1], ":", 2)
	if len(sourceParts) != 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTransferErrorInvalidSource, nil)), nil)
		return dispatcher.EndGroups
	}
	sourceStorageName := sourceParts[0]
	sourcePath := sourceParts[1]

	userID := update.GetUserChat().GetID()

	// Get source storage
	sourceStorage, err := storage.GetStorageByUserIDAndName(ctx, userID, sourceStorageName)
	if err != nil {
		logger.Errorf("Failed to get source storage by user ID and name: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTransferErrorStorageNotFound, map[string]any{
			"StorageName": sourceStorageName,
			"Error":       err,
		})), nil)
		return dispatcher.EndGroups
	}

	// Check if source storage supports listing
	listable, ok := sourceStorage.(storage.StorageListable)
	if !ok {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTransferErrorStorageNotListable, map[string]any{
			"StorageName": sourceStorageName,
		})), nil)
		return dispatcher.EndGroups
	}

	// Check if source storage supports reading
	_, ok = sourceStorage.(storage.StorageReadable)
	if !ok {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTransferErrorStorageNotReadable, map[string]any{
			"StorageName": sourceStorageName,
		})), nil)
		return dispatcher.EndGroups
	}

	// Fetch file list
	replied, err := ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTransferInfoFetchingFiles, nil)), nil)
	if err != nil {
		logger.Errorf("Failed to reply: %s", err)
		return dispatcher.EndGroups
	}

	files, err := listable.ListFiles(ctx, sourcePath)
	if err != nil {
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:      replied.ID,
			Message: i18n.T(i18nk.BotMsgTransferErrorListFilesFailed, map[string]any{"Error": err}),
		})
		return dispatcher.EndGroups
	}

	// Optional filter
	var filter *regexp.Regexp
	if len(args) >= 3 {
		filter, err = regexp.Compile(args[2])
		if err != nil {
			ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
				ID:      replied.ID,
				Message: i18n.T(i18nk.BotMsgTransferErrorInvalidRegex, map[string]any{"Error": err}),
			})
			return dispatcher.EndGroups
		}
	}

	// Filter files
	filteredFiles := make([]storagetypes.FileInfo, 0)
	for _, file := range files {
		if file.IsDir {
			continue
		}
		if filter != nil && !filter.MatchString(file.Name) {
			continue
		}
		filteredFiles = append(filteredFiles, file)
	}

	if len(filteredFiles) == 0 {
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:      replied.ID,
			Message: i18n.T(i18nk.BotMsgTransferErrorNoFilesToTransfer, nil),
		})
		return dispatcher.EndGroups
	}

	// Prepare file paths for callback data
	filePaths := make([]string, 0, len(filteredFiles))
	var totalSize int64
	for _, file := range filteredFiles {
		filePaths = append(filePaths, file.Path)
		totalSize += file.Size
	}

	// Build storage selection keyboard
	markup, err := msgelem.BuildAddSelectStorageKeyboard(storage.GetUserStorages(ctx, userID), tcbdata.Add{
		TaskType:               tasktype.TaskTypeTransfer,
		TransferSourceStorName: sourceStorageName,
		TransferSourcePath:     sourcePath,
		TransferFiles:          filePaths,
	})
	if err != nil {
		logger.Errorf("Failed to build storage selection keyboard: %s", err)
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:      replied.ID,
			Message: i18n.T(i18nk.BotMsgTransferErrorBuildStorageSelectKeyboardFailed, map[string]any{"Error": err}),
		})
		return dispatcher.EndGroups
	}

	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID: replied.ID,
		Message: i18n.T(i18nk.BotMsgTransferInfoFilesSelectStorage, map[string]any{
			"Count":  len(filteredFiles),
			"SizeMB": fmt.Sprintf("%.2f", float64(totalSize)/(1024*1024)),
		}),
		ReplyMarkup: markup,
	})

	return dispatcher.EndGroups
}

func handleTransferCallback(ctx *ext.Context, userID int64, targetStorage storage.Storage, dirPath string, data tcbdata.Add, msgID int) error {
	logger := log.FromContext(ctx)

	// Get source storage
	sourceStorage, err := storage.GetStorageByUserIDAndName(ctx, userID, data.TransferSourceStorName)
	if err != nil {
		logger.Errorf("Failed to get source storage: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      msgID,
			Message: i18n.T(i18nk.BotMsgTransferErrorStorageNotFound, map[string]any{"StorageName": data.TransferSourceStorName, "Error": err}),
		})
		return dispatcher.EndGroups
	}

	// Check if source storage supports listing
	listable, ok := sourceStorage.(storage.StorageListable)
	if !ok {
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      msgID,
			Message: i18n.T(i18nk.BotMsgTransferErrorStorageNotListable, map[string]any{"StorageName": data.TransferSourceStorName}),
		})
		return dispatcher.EndGroups
	}

	// Re-fetch files to get FileInfo (since we only stored paths)
	// This is necessary to get size and other metadata
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:      msgID,
		Message: i18n.T(i18nk.BotMsgTransferInfoFetchingFiles, nil),
	})

	allFiles, err := listable.ListFiles(ctx, data.TransferSourcePath)
	if err != nil {
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      msgID,
			Message: i18n.T(i18nk.BotMsgTransferErrorListFilesFailed, map[string]any{"Error": err}),
		})
		return dispatcher.EndGroups
	}

	// Create a map for quick lookup
	fileMap := make(map[string]storagetypes.FileInfo)
	for _, file := range allFiles {
		fileMap[file.Path] = file
	}

	// Build task elements for the selected files
	elems := make([]transfer.TaskElement, 0, len(data.TransferFiles))
	var totalSize int64
	for _, filePath := range data.TransferFiles {
		fileInfo, ok := fileMap[filePath]
		if !ok {
			logger.Warnf("File not found in source storage: %s", filePath)
			continue
		}
		elem := transfer.NewTaskElement(sourceStorage, fileInfo, targetStorage, dirPath)
		elems = append(elems, *elem)
		totalSize += fileInfo.Size
	}

	if len(elems) == 0 {
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      msgID,
			Message: i18n.T(i18nk.BotMsgTransferErrorNoFilesToTransfer, nil),
		})
		return dispatcher.EndGroups
	}

	// Create and add task
	taskID := xid.New().String()
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	task := transfer.NewTransferTask(
		taskID,
		injectCtx,
		elems,
		transfer.NewProgressTracker(msgID, userID),
		true, // IgnoreErrors
	)

	if err := core.AddTask(injectCtx, task); err != nil {
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      msgID,
			Message: i18n.T(i18nk.BotMsgTransferErrorAddTaskFailed, map[string]any{"Error": err}),
		})
		return dispatcher.EndGroups
	}

	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID: msgID,
		Message: i18n.T(i18nk.BotMsgTransferInfoTaskAdded, map[string]any{
			"Count":  len(elems),
			"SizeMB": fmt.Sprintf("%.2f", float64(totalSize)/(1024*1024)),
			"TaskID": taskID,
		}),
	})

	return dispatcher.EndGroups
}
