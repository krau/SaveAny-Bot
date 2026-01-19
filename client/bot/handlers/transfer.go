package handlers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/batchimport"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func handleTransferCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strutil.ParseArgsRespectQuotes(update.EffectiveMessage.Text)

	if len(args) < 3 {
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

	// Parse target: storage_name:/path
	targetParts := strings.SplitN(args[2], ":", 2)
	if len(targetParts) != 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTransferErrorInvalidTarget, nil)), nil)
		return dispatcher.EndGroups
	}
	targetStorageName := targetParts[0]
	targetPath := targetParts[1]

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

	// Get target storage
	targetStorage, err := storage.GetStorageByUserIDAndName(ctx, userID, targetStorageName)
	if err != nil {
		logger.Errorf("Failed to get target storage by user ID and name: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgTransferErrorTargetNotFound, map[string]any{
			"StorageName": targetStorageName,
			"Error":       err,
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
	if len(args) >= 4 {
		filter, err = regexp.Compile(args[3])
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

	// Create task elements
	elems := make([]batchimport.TaskElement, 0, len(filteredFiles))
	var totalSize int64
	for _, file := range filteredFiles {
		elem := batchimport.NewTaskElement(sourceStorage, file, targetStorage, targetPath)
		elems = append(elems, *elem)
		totalSize += file.Size
	}

	// Create and add task
	taskID := xid.New().String()
	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	task := batchimport.NewBatchImportTask(
		taskID,
		injectCtx,
		elems,
		batchimport.NewProgressTracker(replied.ID, userID),
		true, // IgnoreErrors
	)

	if err := core.AddTask(injectCtx, task); err != nil {
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:      replied.ID,
			Message: i18n.T(i18nk.BotMsgTransferErrorAddTaskFailed, map[string]any{"Error": err}),
		})
		return dispatcher.EndGroups
	}

	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID: replied.ID,
		Message: i18n.T(i18nk.BotMsgTransferInfoTaskAdded, map[string]any{
			"Count":  len(elems),
			"SizeMB": fmt.Sprintf("%.2f", float64(totalSize)/(1024*1024)),
			"TaskID": taskID,
		}),
	})

	return dispatcher.EndGroups
}
