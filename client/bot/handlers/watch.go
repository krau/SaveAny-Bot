package handlers

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"text/template"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/ruleutil"
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/fnamest"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func handleWatchCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchHelpText)), nil)
		return dispatcher.EndGroups
	}
	userChatID := update.GetUserChat().GetID()
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("Failed to get user: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetUserFailed)), nil)
		return dispatcher.EndGroups
	}
	if user.DefaultStorage == "" {
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
		filterType := strings.Split(filterArg, ":")[0]
		filterData := strings.Split(filterArg, ":")[1]
		if filterType == "" || filterData == "" {
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
		UserID: user.ID,
		ChatID: chatID,
		Filter: filter,
	}); err != nil {
		logger.Errorf("Failed to watch chat %d: %s", chatID, err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchErrorWatchChatFailed, map[string]any{"Error": err.Error()})), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchInfoWatchChatStarted, map[string]any{"Chat": chatArg})), nil)
	return dispatcher.EndGroups
}

func handleLswatchCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	userChatID := update.GetUserChat().GetID()
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("Failed to get user: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetUserFailed)), nil)
		return dispatcher.EndGroups
	}
	chats := user.WatchChats
	if len(chats) == 0 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchInfoWatchListEmpty)), nil)
		return dispatcher.EndGroups
	}
	var sb strings.Builder
	sb.WriteString(i18n.T(i18nk.BotMsgWatchInfoWatchListHeader))
	for _, chat := range chats {
		sb.WriteString("- ")
		sb.WriteString(fmt.Sprintf("%d", chat.ChatID))
		if chat.Filter != "" {
			sb.WriteString(i18n.T(i18nk.BotMsgWatchInfoWatchListFilterPrefix))
			sb.WriteString(chat.Filter)
			sb.WriteString(")")
		}
		sb.WriteString("\n")
	}
	ctx.Reply(update, ext.ReplyTextString(sb.String()), nil)
	return dispatcher.EndGroups
}

func handleUnwatchCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchErrorUnwatchNoChatProvided)), nil)
		return dispatcher.EndGroups
	}
	userChatID := update.GetUserChat().GetID()
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("Failed to get user: %s", err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorGetUserFailed)), nil)
		return dispatcher.EndGroups
	}
	chatArg := args[1]
	chatID, err := tgutil.ParseChatID(ctx, chatArg)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgCommonErrorInvalidIdOrUsername, map[string]any{"Error": err.Error()})), nil)
		return dispatcher.EndGroups
	}
	if err := user.UnwatchChat(ctx, chatID); err != nil {
		logger.Errorf("Failed to unwatch chat %d: %s", chatID, err)
		ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchErrorUnwatchChatFailed, map[string]any{"Error": err.Error()})), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString(i18n.T(i18nk.BotMsgWatchInfoWatchChatStopped, map[string]any{"Chat": chatArg})), nil)
	return dispatcher.EndGroups
}

func listenMediaMessageEvent(ch chan userclient.MediaMessageEvent) {
	if userclient.GetCtx() == nil {
		return
	}
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
			var dirPath string
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
