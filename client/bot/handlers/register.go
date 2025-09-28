package handlers

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	sabotfilters "github.com/krau/SaveAny-Bot/client/bot/handlers/utils/filters"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/mediautil"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/re"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/ruleutil"
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/fnamest"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

type DescCommandHandler struct {
	Cmd     string
	Desc    string
	handler func(ctx *ext.Context, u *ext.Update) error
}

var CommandHandlers = []DescCommandHandler{
	{"start", "开始使用", handleHelpCmd},
	{"silent", "切换静默模式", handleSilentCmd},
	{"storage", "设置默认存储端", handleStorageCmd},
	{"dir", "管理存储文件夹", handleDirCmd},
	{"rule", "管理自动存储规则", handleRuleCmd},
	{"watch", "监听聊天(UserBot)", handleWatchCmd},
	{"unwatch", "取消监听聊天(UserBot)", handleUnwatchCmd},
	{"save", "保存文件", handleSilentMode(handleSaveCmd, handleSilentSaveReplied)},
	{"config", "修改配置", handleConfigCmd},
	{"fnametmpl", "设置文件命名模板", handleConfigFnameTmpl},
	{"update", "检查更新", handleUpdateCmd},
	{"help", "显示帮助", handleHelpCmd},
}

func Register(disp dispatcher.Dispatcher) {
	disp.AddHandler(handlers.NewMessage(filters.Message.ChatType(filters.ChatTypeChannel), func(ctx *ext.Context, u *ext.Update) error {
		return dispatcher.EndGroups
	}))
	disp.AddHandler(handlers.NewMessage(filters.Message.ChatType(filters.ChatTypeChat), func(ctx *ext.Context, u *ext.Update) error {
		return dispatcher.EndGroups
	}))
	disp.AddHandler(handlers.NewMessage(filters.Message.All, checkPermission))
	for _, info := range CommandHandlers {
		disp.AddHandler(handlers.NewCommand(info.Cmd, info.handler))
	}
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("update"), handleUpdateCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeAdd), handleAddCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeSetDefault), handleSetDefaultCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeCancel), handleCancelCallback))
	disp.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix(tcbdata.TypeConfig), handleConfigCallback))
	disp.AddHandler(handlers.NewMessage(sabotfilters.RegexUrl(regexp.MustCompile(re.TgMessageLinkRegexString)), handleSilentMode(handleMessageLink, handleSilentSaveLink)))
	disp.AddHandler(handlers.NewMessage(sabotfilters.RegexUrl(regexp.MustCompile(re.TelegraphUrlRegexString)), handleSilentMode(handleTelegraphUrlMessage, handleSilentSaveTelegraph)))
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
			switch user.FilenameStrategy {
			case fnamest.Message.String():
				file.SetName(tgutil.GenFileNameFromMessage(*file.Message()))
			case fnamest.Template.String():
				if user.FilenameTemplate == "" {
					logger.Warnf("Empty filename template for user %d, using default filename", user.ChatID)
					break
				}
				// [TODO] refactor this
				message := file.Message()
				tmpl, err := template.New("filename").Parse(user.FilenameTemplate)
				if err != nil {
					logger.Errorf("Failed to parse filename template for user %d: %s", user.ChatID, err)
					break
				}
				data := mediautil.FilenameTemplateData{
					MsgID: func() string {
						id := message.GetID()
						if id == 0 {
							return ""
						}
						return fmt.Sprintf("%d", id)
					}(),
					MsgTags: func() string {
						tags := strutil.ExtractTagsFromText(message.GetMessage())
						if len(tags) == 0 {
							return ""
						}
						return strings.Join(tags, "_")
					}(),
					MsgGen: tgutil.GenFileNameFromMessage(*message),
					OrigName: func() string {
						f, _ := tgutil.GetMediaFileName(message.Media)
						return f
					}(),
					MsgDate: func() string {
						date := message.GetDate()
						if date == 0 {
							return ""
						}
						t := time.Unix(int64(date), 0)
						return t.Format("2006-01-02_15-04-05")
					}(),
				}.ToMap()
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
