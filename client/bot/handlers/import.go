package handlers

import (
	"fmt"
	"regexp"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	storconfig "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/batchimport"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func handleImportCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strutil.ParseArgsRespectQuotes(update.EffectiveMessage.Text)

	if len(args) < 3 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgImportUsage, nil)), nil)
		return dispatcher.EndGroups
	}

	storageName := args[1]
	dirPath := args[2]

	userID := update.GetUserChat().GetID()

	stor, err := storage.GetStorageByUserIDAndName(ctx, userID, storageName)
	if err != nil {
		logger.Errorf("Failed to get storage by user ID and name: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgImportErrorStorageNotFound, map[string]any{
			"StorageName": storageName,
			"Error":       err,
		})), nil)
		return dispatcher.EndGroups
	}

	listable, ok := stor.(storage.StorageListable)
	if !ok {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgImportErrorStorageNotListable, map[string]any{
			"StorageName": storageName,
		})), nil)
		return dispatcher.EndGroups
	}

	_, ok = stor.(storage.StorageReadable)
	if !ok {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgImportErrorStorageNotReadable, map[string]any{
			"StorageName": storageName,
		})), nil)
		return dispatcher.EndGroups
	}

	telegramStorage, err := storage.GetTelegramStorageByUserID(ctx, userID)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgImportErrorNoTelegramStorage, map[string]any{
			"Error": err,
		})), nil)
		return dispatcher.EndGroups
	}

	replied, err := ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgImportInfoFetchingFiles, nil)), nil)
	if err != nil {
		logger.Errorf("Failed to reply: %s", err)
		return dispatcher.EndGroups
	}

	files, err := listable.ListFiles(ctx, dirPath)
	if err != nil {
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:      replied.ID,
			Message: i18n.T(i18nk.BotMsgImportErrorListFilesFailed, map[string]any{"Error": err}),
		})
		return dispatcher.EndGroups
	}

	var filter *regexp.Regexp
	if len(args) >= 5 {
		filter, err = regexp.Compile(args[4])
		if err != nil {
			ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
				ID:      replied.ID,
				Message: i18n.T(i18nk.BotMsgImportErrorInvalidRegex, map[string]any{"Error": err}),
			})
			return dispatcher.EndGroups
		}
	}

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
			Message: i18n.T(i18nk.BotMsgImportErrorNoFilesToImport, nil),
		})
		return dispatcher.EndGroups
	}

	// Get default chat_id from Telegram storage config
	targetChatID := int64(0)
	if telegramCfg := config.C().GetStorageByName(telegramStorage.Name()); telegramCfg != nil {
		if tgCfg, ok := telegramCfg.(*storconfig.TelegramStorageConfig); ok {
			targetChatID = tgCfg.ChatID
		}
	}

	if len(args) >= 4 {
		parsedChatID, err := tgutil.ParseChatID(ctx, args[3])
		if err != nil {
			ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
				ID:      replied.ID,
				Message: i18n.T(i18nk.BotMsgImportErrorInvalidChatId, map[string]any{"Error": err}),
			})
			return dispatcher.EndGroups
		}
		targetChatID = parsedChatID
	}

	if targetChatID == 0 {
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:      replied.ID,
			Message: i18n.T(i18nk.BotMsgImportErrorNoTargetChatId, nil),
		})
		return dispatcher.EndGroups
	}

	elems := make([]batchimport.TaskElement, 0, len(filteredFiles))
	var totalSize int64
	for _, file := range filteredFiles {
		elem := batchimport.NewTaskElement(stor, file, telegramStorage, targetChatID)
		elems = append(elems, *elem)
		totalSize += file.Size
	}

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
			Message: i18n.T(i18nk.BotMsgImportErrorAddTaskFailed, map[string]any{"Error": err}),
		})
		return dispatcher.EndGroups
	}

	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:      replied.ID,
		Message: i18n.T(i18nk.BotMsgImportInfoTaskAdded, map[string]any{
			"Count":  len(elems),
			"SizeMB": fmt.Sprintf("%.2f", float64(totalSize)/(1024*1024)),
			"TaskID": taskID,
		}),
	})

	return dispatcher.EndGroups
}
