package handlers

import (
	"path"
	"regexp"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/ruleutil"
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func Register(disp dispatcher.Dispatcher) {
	disp.AddHandler(handlers.NewMessage(filters.Message.ChatType(filters.ChatTypeChannel), func(ctx *ext.Context, u *ext.Update) error {
		return dispatcher.EndGroups
	}))
	disp.AddHandler(handlers.NewMessage(filters.Message.ChatType(filters.ChatTypeChat), func(ctx *ext.Context, u *ext.Update) error {
		return dispatcher.EndGroups
	}))
	disp.AddHandler(handlers.NewMessage(filters.Message.All, checkPermission))
	disp.AddHandler(handlers.NewCommand("start", handleHelpCmd))
	disp.AddHandler(handlers.NewCommand("help", handleHelpCmd))
	disp.AddHandler(handlers.NewCommand("silent", handleSilentCmd))
	disp.AddHandler(handlers.NewCommand("storage", handleStorageCmd))
	disp.AddHandler(handlers.NewCommand("dir", handleDirCmd))
	disp.AddHandler(handlers.NewCommand("rule", handleRuleCmd))
	disp.AddHandler(handlers.NewCommand("watch", handleWatchCmd))
	disp.AddHandler(handlers.NewCommand("unwatch", handleUnwatchCmd))
	disp.AddHandler(handlers.NewCommand("save", handleSilentMode(handleSaveCmd, handleSilentSaveReplied)))
	disp.AddHandler(handlers.NewCommand("config", handleConfigCmd))
	disp.AddHandler(handlers.NewCommand("update", handleUpdateCmd))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("update"), handleUpdateCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeAdd), handleAddCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeSetDefault), handleSetDefaultCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeCancel), handleCancelCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeConfig), handleConfigCallback))
	linkRegexFilter, err := filters.Message.Regex(re.TgMessageLinkRegexString)
	if err != nil {
		panic("failed to create regex filter: " + err.Error())
	}
	disp.AddHandler(handlers.NewMessage(linkRegexFilter, handleSilentMode(handleMessageLink, handleSilentSaveLink)))
	telegraphUrlRegexFilter, err := filters.Message.Regex(re.TelegraphUrlRegexString)
	if err != nil {
		panic("failed to create Telegraph URL regex filter: " + err.Error())
	}
	disp.AddHandler(handlers.NewMessage(telegraphUrlRegexFilter, handleSilentMode(handleTelegraphUrlMessage, handleSilentSaveTelegraph)))
	disp.AddHandler(handlers.NewMessage(filters.Message.Media, handleSilentMode(handleMediaMessage, handleSilentSaveMedia)))
	disp.AddHandler(handlers.NewMessage(filters.Message.Text, handleSilentMode(handleTextMessage, handleSilentSaveText)))

	if config.C().Telegram.Userbot.Enable {
		go listenMediaMessageEvent(userclient.GetMediaMessageCh())
	}
}

func listenMediaMessageEvent(ch chan userclient.MediaMessageEvent) {
	logger := log.FromContext(userclient.GetCtx())
	for event := range ch {
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
			if chat.Filter != "" {
				filter := strings.Split(chat.Filter, ":")
				if len(filter) != 2 {
					logger.Warnf("Invalid filter format in chat %d, skipping", chat.ChatID)
					continue
				}
				filterType := filter[0]
				filterData := filter[1]
				switch filterType {
				case "msgre": // [TODO] enums for filter types
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
			if user.DefaultStorage == "" {
				logger.Warnf("User %d has no default storage set, skipping media message handling", chat.UserID)
				continue
			}
			stor, err := storage.GetStorageByUserIDAndName(ctx, user.ChatID, user.DefaultStorage)
			if err != nil {
				logger.Errorf("Failed to get storage by user ID %d and name %s: %v", user.ChatID, user.DefaultStorage, err)
				continue
			}
			var dirPath string
			if user.ApplyRule && user.Rules != nil {
				matched, matchedStorageName, matchedDirPath := ruleutil.ApplyRule(ctx, user.Rules, ruleutil.NewInput(file))
				if !matched {
					goto startCreateTask
				}
				dirPath = matchedDirPath.String()
				if matchedStorageName.IsUsable() {
					stor, err = storage.GetStorageByUserIDAndName(ctx, user.ChatID, matchedStorageName.String())
					if err != nil {
						logger.Errorf("Failed to get storage by user ID and name: %s", err)
						continue
					}
				}
			}
		startCreateTask:
			storagePath := stor.JoinStoragePath(path.Join(dirPath, file.Name()))
			injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
			taskid := xid.New().String()
			task, err := tfile.NewTGFileTask(taskid, injectCtx, file, stor, storagePath, nil)
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
