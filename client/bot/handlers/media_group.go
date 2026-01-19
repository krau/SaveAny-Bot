package handlers

import (
	"sync"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

type MediaGroupHandler struct {
	groups    map[int64][]tfile.TGFileMessage
	timers    map[int64]*time.Timer
	mu        sync.Mutex
	timeout   time.Duration
	setupOnce sync.Once
}

func (m *MediaGroupHandler) SetupTimeout(timeoutSec int) {
	m.setupOnce.Do(func() {
		if timeoutSec < 1 {
			timeoutSec = 1
		}
		m.timeout = time.Duration(timeoutSec) * time.Second
	})
}

var (
	mediaGroupHandler = &MediaGroupHandler{
		groups: make(map[int64][]tfile.TGFileMessage),
		timers: make(map[int64]*time.Timer),
		mu:     sync.Mutex{},
	}
)

func handleGroupMediaMessage(ctx *ext.Context, update *ext.Update, message *tg.Message, groupID int64) error {
	mediaGroupHandler.SetupTimeout(max(config.C().Telegram.MediaGroupTimeout, 1))
	logger := log.FromContext(ctx)
	media := message.Media
	supported := mediautil.IsSupported(media)
	if !supported {
		return dispatcher.EndGroups
	}
	file, err := tfile.FromMediaMessage(media, ctx.Raw, message, tfile.WithNameIfEmpty(
		tgutil.GenFileNameFromMessage(*message),
	))
	if err != nil {
		logger.Errorf("Failed to get file from media: %s", err)
		return dispatcher.EndGroups
	}
	mediaGroupHandler.mu.Lock()
	defer mediaGroupHandler.mu.Unlock()
	if mediaGroupHandler.groups[groupID] == nil {
		mediaGroupHandler.groups[groupID] = make([]tfile.TGFileMessage, 0)
	}
	mediaGroupHandler.groups[groupID] = append(mediaGroupHandler.groups[groupID], file)

	if timer, exists := mediaGroupHandler.timers[groupID]; exists {
		timer.Stop()
	}
	mediaGroupHandler.timers[groupID] = time.AfterFunc(mediaGroupHandler.timeout, func() {
		processMediaGroup(ctx, update, groupID)
	})
	return dispatcher.EndGroups
}

func processMediaGroup(ctx *ext.Context, update *ext.Update, groupID int64) {
	logger := log.FromContext(ctx)
	mediaGroupHandler.mu.Lock()
	items := mediaGroupHandler.groups[groupID]
	delete(mediaGroupHandler.groups, groupID)
	delete(mediaGroupHandler.timers, groupID)
	mediaGroupHandler.mu.Unlock()
	if len(items) == 0 {
		logger.Warn("No media items to process for group", "groupID", groupID)
		return
	}
	logger.Debugf("Processing media group %d with %d items", groupID, len(items))

	userId := update.GetUserChat().GetID()
	msg, err := ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgMediaGroupInfoSavingFiles, nil)), nil)
	if err != nil {
		logger.Errorf("Failed to reply: %s", err)
		return
	}
	stor := storage.FromContext(ctx)
	if stor != nil {
		// In silent mode
		if len(items) == 1 {
			shortcut.CreateAndAddTGFileTaskWithEdit(ctx, userId, stor, "", items[0], msg.ID)
			return
		}
		shortcut.CreateAndAddBatchTGFileTaskWithEdit(ctx, userId, stor, "", items, msg.ID)
		return
	}

	stors := storage.GetUserStorages(ctx, userId)
	markup, err := msgelem.BuildAddSelectStorageKeyboard(stors, tcbdata.Add{
		Files:   items,
		AsBatch: len(items) > 1,
	})
	if err != nil {
		logger.Errorf("Failed to build storage selection keyboard: %s", err)
		ctx.EditMessage(userId, &tg.MessagesEditMessageRequest{
			ID: msg.ID,
			Message: i18n.T(i18nk.BotMsgMediaGroupErrorBuildStorageSelectKeyboardFailed, map[string]any{
				"Error": err.Error(),
			}),
		})
		return
	}
	ctx.EditMessage(userId, &tg.MessagesEditMessageRequest{
		ID: msg.ID,
		Message: i18n.T(i18nk.BotMsgMediaGroupInfoGroupFoundFilesSelectStorage, map[string]any{
			"Count": len(items),
		}),
		ReplyMarkup: markup,
	})
}
