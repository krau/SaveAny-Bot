package handlers

import (
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/mediafilter"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
)

// mediaGroupKey uniquely identifies a media group by chat, sender, and group
// ID so files from different users in the same chat can never be mixed.
type mediaGroupKey struct {
	chatID  int64
	userID  int64
	groupID int64
}

type MediaGroupHandler struct {
	groups    map[mediaGroupKey][]tfile.TGFileMessage
	timers    map[mediaGroupKey]*time.Timer
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
		groups: make(map[mediaGroupKey][]tfile.TGFileMessage),
		timers: make(map[mediaGroupKey]*time.Timer),
		mu:     sync.Mutex{},
	}
)

func handleGroupMediaMessage(ctx *ext.Context, update *ext.Update, message *tg.Message, groupID int64) error {
	mediaGroupHandler.SetupTimeout(max(config.C().Telegram.MediaGroupTimeout, 1))
	logger := log.FromContext(ctx)
	media := message.Media
	if !mediautil.IsSupported(media) || !mediafilter.Allowed(media) {
		return dispatcher.EndGroups
	}
	userId := responsibleUserID(update)
	userDB, err := database.GetUserByChatID(ctx, userId)
	if err != nil {
		return err
	}
	tfOpts := mediautil.TfileOptions(ctx, userDB, message)
	file, err := tfile.FromMediaMessage(media, ctx.Raw, message, tfOpts...)
	if err != nil {
		logger.Errorf("Failed to get file from media: %s", err)
		return dispatcher.EndGroups
	}
	mediaGroupHandler.mu.Lock()
	defer mediaGroupHandler.mu.Unlock()
	key := mediaGroupKey{
		chatID:  update.EffectiveChat().GetID(),
		userID:  userId,
		groupID: groupID,
	}
	if mediaGroupHandler.groups[key] == nil {
		mediaGroupHandler.groups[key] = make([]tfile.TGFileMessage, 0)
	}
	mediaGroupHandler.groups[key] = append(mediaGroupHandler.groups[key], file)

	if timer, exists := mediaGroupHandler.timers[key]; exists {
		timer.Stop()
	}
	mediaGroupHandler.timers[key] = time.AfterFunc(mediaGroupHandler.timeout, func() {
		processMediaGroup(ctx, update, key)
	})
	return dispatcher.EndGroups
}

func processMediaGroup(ctx *ext.Context, update *ext.Update, key mediaGroupKey) {
	logger := log.FromContext(ctx)
	mediaGroupHandler.mu.Lock()
	items := mediaGroupHandler.groups[key]
	delete(mediaGroupHandler.groups, key)
	delete(mediaGroupHandler.timers, key)
	mediaGroupHandler.mu.Unlock()
	if len(items) == 0 {
		logger.Warn("No media items to process for group", "groupID", key.groupID)
		return
	}
	logger.Debugf("Processing media group %d with %d items", key.groupID, len(items))

	userId := responsibleUserID(update)
	msg, err := ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgMediaGroupInfoSavingFiles, nil)), nil)
	if err != nil {
		logger.Errorf("Failed to reply: %s", err)
		return
	}
	stor := storage.FromContext(ctx)
	if stor != nil {
		// In silent mode
		if len(items) == 1 {
			shortcut.CreateAndAddTGFileTaskWithEdit(ctx, userId, key.chatID, stor, "", items[0], msg.ID)
			return
		}
		shortcut.CreateAndAddBatchTGFileTaskWithEdit(ctx, userId, key.chatID, stor, "", items, msg.ID)
		return
	}

	stors := storage.GetUserStorages(ctx, userId)
	markup, err := msgelem.BuildAddSelectStorageKeyboard(stors, tcbdata.Add{
		Files:   items,
		AsBatch: len(items) > 1,
	})
	if err != nil {
		logger.Errorf("Failed to build storage selection keyboard: %s", err)
		ctx.EditMessage(key.chatID, &tg.MessagesEditMessageRequest{
			ID: msg.ID,
			Message: i18n.T(i18nk.BotMsgMediaGroupErrorBuildStorageSelectKeyboardFailed, map[string]any{
				"Error": err.Error(),
			}),
		})
		return
	}
	ctx.EditMessage(key.chatID, &tg.MessagesEditMessageRequest{
		ID: msg.ID,
		Message: i18n.T(i18nk.BotMsgMediaGroupInfoGroupFoundFilesSelectStorage, map[string]any{
			"Count": len(items),
		}),
		ReplyMarkup: markup,
	})
}
