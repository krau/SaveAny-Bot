package handlers

import (
	"fmt"
	"sync"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/shortcut"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/fnamest"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleMediaMessage(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	message := update.EffectiveMessage.Message
	groupID, isGroup := message.GetGroupedID()
	if isGroup && groupID != 0 {
		return handleGroupMediaMessage(ctx, update, message, groupID)
	}
	logger.Debugf("Got media: %s", message.Media.TypeName())
	userId := update.GetUserChat().GetID()
	userDB, err := database.GetUserByChatID(ctx, userId)
	if err != nil {
		return err
	}
	tfOpts := make([]tfile.TGFileOption, 0)
	switch userDB.FilenameStrategy {
	case fnamest.Message.String():
		tfOpts = append(tfOpts, tfile.WithName(tgutil.GenFileNameFromMessage(*message)))
	default:
	}
	msg, file, err := shortcut.GetFileFromMessageWithReply(ctx, update, message, tfOpts...)
	if err != nil {
		return err
	}

	stors := storage.GetUserStorages(ctx, userId)
	req, err := msgelem.BuildAddOneSelectStorageMessage(ctx, stors, file, msg.ID)
	if err != nil {
		logger.Errorf("构建存储选择消息失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("构建存储选择消息失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	ctx.EditMessage(update.EffectiveChat().GetID(), req)
	return dispatcher.EndGroups
}

func handleSilentSaveMedia(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	stor := storage.FromContext(ctx)
	if stor == nil {
		logger.Warn("Context storage is nil")
		ctx.Reply(update, ext.ReplyTextString("未找到存储"), nil)
		return dispatcher.EndGroups
	}
	message := update.EffectiveMessage.Message
	groupID, isGroup := message.GetGroupedID()
	if isGroup && groupID != 0 {
		return handleGroupMediaMessage(ctx, update, message, groupID)
	}
	logger.Debugf("Got media: %s", message.Media.TypeName())
	userID := update.GetUserChat().GetID()
	userDB, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	tfOpts := make([]tfile.TGFileOption, 0)
	switch userDB.FilenameStrategy {
	case fnamest.Message.String():
		tfOpts = append(tfOpts, tfile.WithName(tgutil.GenFileNameFromMessage(*message)))
	default:
	}
	msg, file, err := shortcut.GetFileFromMessageWithReply(ctx, update, message, tfOpts...)
	if err != nil {
		return err
	}
	return shortcut.CreateAndAddTGFileTaskWithEdit(ctx, userID, stor, "", file, msg.ID)
}

type MediaGroupHandler struct {
	groups  map[int64][]tfile.TGFileMessage
	timers  map[int64]*time.Timer
	mu      sync.Mutex
	timeout time.Duration
}

var mediaGroupHandler = &MediaGroupHandler{
	groups:  make(map[int64][]tfile.TGFileMessage),
	timers:  make(map[int64]*time.Timer),
	timeout: 1 * time.Second,
}

func handleGroupMediaMessage(ctx *ext.Context, update *ext.Update, message *tg.Message, groupID int64) error {
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
	msg, err := ctx.Reply(update, ext.ReplyTextString("正在保存文件..."), nil)
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
		logger.Errorf("构建存储选择键盘失败: %s", err)
		ctx.EditMessage(userId, &tg.MessagesEditMessageRequest{
			ID:      msg.ID,
			Message: "构建存储选择键盘失败: " + err.Error(),
		})
		return
	}
	ctx.EditMessage(userId, &tg.MessagesEditMessageRequest{
		ID:          msg.ID,
		Message:     fmt.Sprintf("共 %d 个文件, 请选择存储位置", len(items)),
		ReplyMarkup: markup,
	})
}
