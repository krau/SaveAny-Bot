package handlers

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/ruleutil"
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	coretfile "github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/fnamest"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/dispatcher"
	"github.com/krau/mygotg/ext"
	"github.com/rs/xid"
)

func handleWatchCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(watchMenuText("")), &ext.ReplyOpts{Markup: watchMenuMarkup()})
		return dispatcher.EndGroups
	}
	userChatID := responsibleUserID(update)
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("Failed to get user: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetUserFailed)), nil)
		return dispatcher.EndGroups
	}
	if user.DefaultStorageID == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorDefaultStorageNotSet)), nil)
		return dispatcher.EndGroups
	}
	chatArg := args[1]
	chatID, err := tgutil.ParseChatID(ctx, chatArg)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorInvalidIdOrUsername, map[string]any{"Error": err.Error()})), nil)
		return dispatcher.EndGroups
	}
	watching, err := user.WatchingChat(ctx, chatID)
	if err != nil {
		logger.Errorf("Failed to check if user is watching chat %d: %s", chatID, err)
		return dispatcher.EndGroups
	}
	if watching {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchInfoAlreadyWatchingChat)), nil)
		return dispatcher.EndGroups
	}
	filter := ""
	if len(args) > 2 {
		filterArg := strings.Join(args[2:], " ")
		filterType, filterData, ok := strings.Cut(filterArg, ":")
		if !ok || filterType == "" || filterData == "" {
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchErrorFilterFormatInvalid)), nil)
			return dispatcher.EndGroups
		}
		switch filterType {
		case "msgre":
			_, err := regexp.Compile(filterData)
			if err != nil {
				ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorInvalidRegex, map[string]any{"Error": err.Error()})), nil)
				return dispatcher.EndGroups
			}
			filter = filterType + ":" + filterData
		default:
			ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchErrorFilterTypeUnsupported)), nil)
			return dispatcher.EndGroups
		}
	}
	if err := user.WatchChat(ctx, database.WatchChat{
		UserID:        user.ID,
		ChatID:        chatID,
		MessageFilter: filter,
	}); err != nil {
		logger.Errorf("Failed to watch chat %d: %s", chatID, err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchErrorWatchChatFailed, map[string]any{"Error": err.Error()})), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchInfoWatchChatStarted, map[string]any{"Chat": chatArg})), nil)
	return dispatcher.EndGroups
}

func watchMenuText(extra string) string {
	text := i18n.T(i18nk.BotMsgWatchMenuTitle) + "\n\n" + i18n.T(i18nk.BotMsgWatchMenuHint)
	if extra != "" {
		text += "\n\n" + extra
	}
	return text
}

func watchMenuMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{Rows: []tg.KeyboardInlineButtonRow{
		{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgWatchMenuAdd), fmt.Appendf(nil, "%s add", tcbdata.TypeWatch)),
			markup.Callback(i18n.T(i18nk.BotMsgWatchMenuList), fmt.Appendf(nil, "%s list", tcbdata.TypeWatch)),
		}},
		{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgWatchMenuNotify), fmt.Appendf(nil, "%s notify", tcbdata.TypeWatch)),
			markup.Callback(i18n.T(i18nk.BotMsgWatchMenuSync), fmt.Appendf(nil, "%s sync", tcbdata.TypeWatch)),
		}},
	}}
}

func watchBackMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{Rows: []tg.KeyboardInlineButtonRow{{
		Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgWatchMenuBack), fmt.Appendf(nil, "%s menu", tcbdata.TypeWatch)),
		},
	}}}
}

func handleWatchCallback(ctx *ext.Context, update *ext.Update) error {
	args := strings.Fields(string(update.CallbackQuery.Data))
	if len(args) < 2 {
		return fmt.Errorf("invalid callback data: %q", update.CallbackQuery.Data)
	}
	switch args[1] {
	case "menu":
		return editWatchMenu(ctx, update)
	case "add":
		return editWatchAdd(ctx, update)
	case "list":
		return editWatchList(ctx, update)
	case "notify":
		return editWatchNotify(ctx, update)
	case "sync":
		return editWatchSync(ctx, update)
	case "rm":
		return editWatchRemove(ctx, update, args)
	default:
		return editWatchMenu(ctx, update)
	}
}

func editWatchMenu(ctx *ext.Context, update *ext.Update) error {
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     watchMenuText(""),
		ReplyMarkup: watchMenuMarkup(),
	})
	return dispatcher.EndGroups
}

func editWatchAdd(ctx *ext.Context, update *ext.Update) error {
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     watchMenuText(i18n.T(i18nk.BotMsgWatchMenuAddHelp)),
		ReplyMarkup: watchBackMarkup(),
	})
	return dispatcher.EndGroups
}

func watchListText(chats []database.WatchChat) string {
	if len(chats) == 0 {
		return i18n.T(i18nk.BotMsgWatchInfoWatchListEmpty)
	}
	var b strings.Builder
	b.WriteString(i18n.T(i18nk.BotMsgWatchInfoWatchListHeader))
	for _, chat := range chats {
		fmt.Fprintf(&b, "\n- %d", chat.ChatID)
		if chat.MessageFilter != "" {
			b.WriteString(i18n.T(i18nk.BotMsgWatchInfoWatchListFilterPrefix))
			b.WriteString(chat.MessageFilter)
			b.WriteString(")")
		}
	}
	return b.String()
}

func watchListMarkup(chats []database.WatchChat) *tg.ReplyInlineMarkup {
	rows := make([]tg.KeyboardInlineButtonRow, 0, len(chats)+1)
	for _, chat := range chats {
		rows = append(rows, tg.KeyboardInlineButtonRow{Buttons: []tg.KeyboardInlineButton{
			markup.Callback(i18n.T(i18nk.BotMsgWatchMenuStop), fmt.Appendf(nil, "%s rm %d", tcbdata.TypeWatch, chat.ChatID)),
		}})
	}
	rows = append(rows, watchBackMarkup().Rows...)
	return &tg.ReplyInlineMarkup{Rows: rows}
}

func editWatchList(ctx *ext.Context, update *ext.Update) error {
	userID := update.CallbackQuery.GetUserID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     watchListText(user.WatchChats),
		ReplyMarkup: watchListMarkup(user.WatchChats),
	})
	return dispatcher.EndGroups
}

func editWatchNotify(ctx *ext.Context, update *ext.Update) error {
	userID := update.CallbackQuery.GetUserID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	user.WatchNotify = !user.WatchNotify
	if err := database.UpdateUser(ctx, user); err != nil {
		return err
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     watchMenuText(i18n.T(watchNotifyStateKey(user.WatchNotify))),
		ReplyMarkup: watchMenuMarkup(),
	})
	return dispatcher.EndGroups
}

func editWatchSync(ctx *ext.Context, update *ext.Update) error {
	count, err := syncPeers(ctx)
	if err != nil {
		ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
			ID:      update.CallbackQuery.GetMsgID(),
			Message: i18n.T(i18nk.BotMsgSyncpeersFailed, map[string]any{"Error": err.Error()}),
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(chatID(update), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     watchMenuText(i18n.T(i18nk.BotMsgSyncpeersSuccess, map[string]any{"Count": count})),
		ReplyMarkup: watchMenuMarkup(),
	})
	return dispatcher.EndGroups
}

func editWatchRemove(ctx *ext.Context, update *ext.Update, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("invalid callback data: %q", update.CallbackQuery.Data)
	}
	chatID, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return err
	}
	userID := update.CallbackQuery.GetUserID()
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		return err
	}
	if err := user.UnwatchChat(ctx, chatID); err != nil {
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(update.CallbackQuery.GetQueryID(), i18n.T(i18nk.BotMsgWatchErrorUnwatchChatFailed, map[string]any{"Error": err.Error()})))
		return dispatcher.EndGroups
	}
	return editWatchList(ctx, update)
}

type watchMediaGroupHandler struct {
	groups map[int64]map[uint][]tfile.TGFileMessage // chatID -> userID -> files
	timers map[int64]map[uint]*time.Timer
	mu     sync.Mutex
}

var watchMediaGroupMgr = &watchMediaGroupHandler{
	groups: make(map[int64]map[uint][]tfile.TGFileMessage),
	timers: make(map[int64]map[uint]*time.Timer),
}

func (w *watchMediaGroupHandler) addFile(chatID int64, userID uint, file tfile.TGFileMessage, timeout time.Duration, callback func([]tfile.TGFileMessage)) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.groups[chatID] == nil {
		w.groups[chatID] = make(map[uint][]tfile.TGFileMessage)
	}
	if w.timers[chatID] == nil {
		w.timers[chatID] = make(map[uint]*time.Timer)
	}

	if timer, exists := w.timers[chatID][userID]; exists {
		timer.Stop()
	}

	w.groups[chatID][userID] = append(w.groups[chatID][userID], file)

	w.timers[chatID][userID] = time.AfterFunc(timeout, func() {
		w.mu.Lock()
		files := w.groups[chatID][userID]
		delete(w.groups[chatID], userID)
		delete(w.timers[chatID], userID)
		if len(w.groups[chatID]) == 0 {
			delete(w.groups, chatID)
		}
		if len(w.timers[chatID]) == 0 {
			delete(w.timers, chatID)
		}
		w.mu.Unlock()

		if len(files) > 0 {
			callback(files)
		}
	})
}

// ListenUserbotMediaEvents consumes userbot media events until the channel is
// closed. resolve returns the client context used for notifications.
func ListenUserbotMediaEvents(runCtx context.Context, resolve func() *ext.Context) {
	logger := log.FromContext(runCtx)
	for event := range userclient.GetMediaMessageCh() {
		if event.File == nil || event.Ctx == nil {
			continue
		}
		botCtx := resolve()
		if botCtx == nil {
			continue
		}
		logger.Debug("Received media message event", "chat_id", event.ChatID, "file_name", event.File.Name())
		ctx := event.Ctx
		file := event.File
		chats, err := database.GetWatchChatsByChatID(ctx, event.ChatID)
		if err != nil {
			logger.Errorf("Failed to get watch chats for chat ID %d: %v", event.ChatID, err)
			continue
		}
		msgText := event.File.Message().GetMessage()
		for _, chat := range chats {
			if chat.MessageFilter != "" {
				filter := strings.Split(chat.MessageFilter, ":")
				if len(filter) != 2 {
					logger.Warnf("Invalid filter format in chat %d, skipping", chat.ChatID)
					continue
				}
				filterType := filter[0]
				filterData := filter[1]
				switch filterType {
				case "msgre":
					if ok, err := regexp.MatchString(filterData, msgText); err != nil {
						continue
					} else if !ok {
						continue
					}
				default:
					logger.Warnf("Unsupported filter type %s in chat %d, skipping", filterType, chat.ChatID)
					continue
				}
			}
			user, err := database.GetUserByID(ctx, chat.UserID)
			if err != nil {
				logger.Errorf("Failed to get user by ID %d: %v", chat.UserID, err)
				continue
			}
			if user.DefaultStorageID == 0 {
				logger.Warnf("User %d has no default storage set, skipping media message handling", chat.UserID)
				continue
			}
			stor, err := storage.GetStorageByUserIDAndID(ctx, user.ChatID, user.DefaultStorageID)
			if err != nil {
				logger.Errorf("Failed to get storage ID %d for user %d: %v", user.DefaultStorageID, user.ChatID, err)
				continue
			}
			var defaultDirPath string
			if user.DefaultDirID != 0 {
				dir, err := database.GetDirByID(ctx, user.DefaultDirID)
				if err != nil {
					logger.Warnf("Failed to get default dir for user %d: %v, using root", user.ChatID, err)
				} else {
					defaultDirPath = dir.Path
				}
			}
			switch user.FilenameStrategy {
			case fnamest.Message.String():
				file.SetName(tgutil.GenFileNameFromMessage(*file.Message()))
			case fnamest.Template.String():
				if user.FilenameTemplate == "" {
					logger.Warnf("Empty filename template for user %d, using default filename", user.ChatID)
					break
				}
				message := file.Message()
				tmpl, err := template.New("filename").Parse(user.FilenameTemplate)
				if err != nil {
					logger.Errorf("Failed to parse filename template for user %d: %s", user.ChatID, err)
					break
				}
				data := mediautil.BuildFilenameTemplateData(message)
				var sb strings.Builder
				err = tmpl.Execute(&sb, data)
				if err != nil {
					log.FromContext(ctx).Errorf("failed to execute filename template: %s", err)
					break
				}
				file.SetName(sb.String())
			}

			groupID, isGroup := file.Message().GetGroupedID()
			needAlbumHandling := false
			if isGroup && groupID != 0 && user.ApplyRule && user.Rules != nil {
				_, _, matchedDirPath := ruleutil.ApplyRule(ctx, user.Rules, ruleutil.NewInput(file))
				needAlbumHandling = matchedDirPath.NeedNewForAlbum()
			}

			if needAlbumHandling {
				// For media groups with NEW-FOR-ALBUM rule, collect all files of the same group
				watchMediaGroupMgr.addFile(event.ChatID, user.ID, file, time.Duration(max(config.C().Telegram.MediaGroupTimeout, 1))*time.Second, func(files []tfile.TGFileMessage) {
					processWatchMediaGroup(ctx, botCtx, user, stor, defaultDirPath, files)
				})
				continue
			}

			dirPath := defaultDirPath
			if user.ApplyRule && user.Rules != nil {
				matched, matchedStorageName, matchedDirPath := ruleutil.ApplyRule(ctx, user.Rules, ruleutil.NewInput(file))
				if !matched {
					goto startCreateTask
				}
				dirPath = matchedDirPath.String()
				if matchedStorageName.Usable() {
					stor, err = storage.GetStorageByUserIDAndName(ctx, user.ChatID, matchedStorageName.String())
					if err != nil {
						logger.Errorf("Failed to get storage by user ID and name: %s", err)
						continue
					}
				}
			}
		startCreateTask:
			storagePath := path.Join(dirPath, file.Name())
			injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
			taskid := xid.New().String()
			task, err := coretfile.NewTGFileTask(taskid, injectCtx, file, stor, storagePath, newWatchNotifyProgress(ctx, botCtx, user))
			if err != nil {
				logger.Errorf("create task failed: %s", err)
				continue
			}
			if err := core.AddTask(injectCtx, task); err != nil {
				logger.Errorf("add task failed: %s", err)
				continue
			}
			logger.Infof("Added media message task for user %d in chat %d: %s", chat.UserID, event.ChatID, file.Name())
		}
	}
}

func processWatchMediaGroup(ctx *ext.Context, botCtx *ext.Context, user *database.User, stor storage.Storage, dirPath string, files []tfile.TGFileMessage) {
	logger := log.FromContext(ctx)
	if len(files) == 0 {
		return
	}

	useRule := user.ApplyRule && user.Rules != nil
	files = mediautil.ApplyAlbumCaptions(ctx, user, files)

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

	type albumFile struct {
		file    tfile.TGFileMessage
		storage storage.Storage
		dirPath string
	}
	albumFiles := make(map[int64][]albumFile)

	for _, file := range files {
		storName, ruleDirPath := applyRule(file)
		fileStor := stor
		if storName != stor.Name() && storName != "" {
			var err error
			fileStor, err = storage.GetStorageByUserIDAndName(ctx, user.ChatID, storName)
			if err != nil {
				logger.Errorf("Failed to get storage by user ID and name: %s", err)
				continue
			}
		}

		groupId, isGroup := file.Message().GetGroupedID()
		if !isGroup || groupId == 0 {
			logger.Warnf("File %s is not in a group, skipping", file.Name())
			continue
		}

		// Use the effective dirPath: if rule returns NEW-FOR-ALBUM sentinel, fall back to the
		// base dirPath passed in (which is defaultDirPath from the caller).
		effectiveDirPath := string(ruleDirPath)
		if ruleDirPath.NeedNewForAlbum() {
			effectiveDirPath = dirPath
		}

		if _, ok := albumFiles[groupId]; !ok {
			albumFiles[groupId] = make([]albumFile, 0)
		}
		albumFiles[groupId] = append(albumFiles[groupId], albumFile{
			file:    file,
			storage: fileStor,
			dirPath: effectiveDirPath,
		})
	}

	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	totalTasks := 0
	for groupID, afiles := range albumFiles {
		if len(afiles) == 1 {
			af := afiles[0]
			taskid := xid.New().String()
			task, err := coretfile.NewTGFileTask(taskid, injectCtx, af.file, af.storage, path.Join(af.dirPath, af.file.Name()), newWatchNotifyProgress(ctx, botCtx, user))
			if err != nil {
				logger.Errorf("create task failed for album file: %s", err)
				continue
			}
			if err := core.AddTask(injectCtx, task); err != nil {
				logger.Errorf("add task failed: %s", err)
				continue
			}
			totalTasks++
			continue
		}

		// Use first file's name (without extension) as album folder name
		albumDir := strings.TrimSuffix(path.Base(afiles[0].file.Name()), path.Ext(afiles[0].file.Name()))
		albumStor := afiles[0].storage

		logger.Infof("Creating album folder for group %d: %s with %d files", groupID, albumDir, len(afiles))

		for _, af := range afiles {
			afstorPath := path.Join(af.dirPath, albumDir, af.file.Name())
			taskid := xid.New().String()
			task, err := coretfile.NewTGFileTask(taskid, injectCtx, af.file, albumStor, afstorPath, newWatchNotifyProgress(ctx, botCtx, user))
			if err != nil {
				logger.Errorf("create task failed for album file: %s", err)
				continue
			}
			if err := core.AddTask(injectCtx, task); err != nil {
				logger.Errorf("add task failed: %s", err)
				continue
			}
			totalTasks++
		}
	}
	logger.Infof("Added %d watch media tasks for user %d", totalTasks, user.ChatID)
}
