package bot

import (
"fmt"
"strconv"
"strings"

"github.com/duke-git/lancet/v2/slice"
"github.com/gotd/td/telegram/message/entity"
"github.com/gotd/td/telegram/message/styling"
"github.com/gotd/td/tg"

"github.com/celestix/gotgproto/dispatcher"
"github.com/celestix/gotgproto/dispatcher/handlers"
"github.com/celestix/gotgproto/dispatcher/handlers/filters"
"github.com/celestix/gotgproto/ext"
"github.com/krau/SaveAny-Bot/config"
"github.com/krau/SaveAny-Bot/dao"
"github.com/krau/SaveAny-Bot/logger"
"github.com/krau/SaveAny-Bot/queue"
"github.com/krau/SaveAny-Bot/storage"
"github.com/krau/SaveAny-Bot/types"
)

func RegisterHandlers(dispatcher dispatcher.Dispatcher) {
dispatcher.AddHandler(handlers.NewMessage(filters.Message.All, checkPermission))
dispatcher.AddHandler(handlers.NewCommand("start", start))
dispatcher.AddHandler(handlers.NewCommand("help", help))
dispatcher.AddHandler(handlers.NewCommand("silent", silent))
dispatcher.AddHandler(handlers.NewCommand("storage", storageCmd))
dispatcher.AddHandler(handlers.NewCommand("save", saveCmd))
linkRegexFilter, err := filters.Message.Regex(linkRegexString)
if err != nil {
logger.L.Panicf("Failed to create regular expression filter: %s", err)
}
dispatcher.AddHandler(handlers.NewMessage(linkRegexFilter, handleLinkMessage))
dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("add"), AddToQueue))
dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("set_default"), setDefaultStorage))
dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, handleFileMessage))
}

const noPermissionText string = `
You are not in the whitelist and cannot use this Bot.
You can deploy your own instance: https://github.com/krau/SaveAny-Bot
`

func checkPermission(ctx *ext.Context, update *ext.Update) error {
userID := update.GetUserChat().GetID()
if !slice.Contain(config.Cfg.GetUsersID(), userID) {
ctx.Reply(update, ext.ReplyTextString(noPermissionText), nil)
return dispatcher.EndGroups
}
return dispatcher.ContinueGroups
}

func start(ctx *ext.Context, update *ext.Update) error {
if err := dao.CreateUser(update.GetUserChat().GetID()); err != nil {
logger.L.Errorf("Failed to create user: %s", err)
return dispatcher.EndGroups
}
return help(ctx, update)
}

const helpText string = `
Save Any Bot - Save your Telegram files
Order:
/start - Start using
/help - Display help
/silent - Toggle silent mode
/storage - Sets the default storage location
/save [custom file name] - save the file

Silent mode: When enabled, Bot saves received files directly to the default location without asking again

Default storage location: The location where files are saved in silent mode

Send (forward) a file to the Bot, or send a public channel message link to save the file
`

func help(ctx *ext.Context, update *ext.Update) error {
ctx.Reply(update, ext.ReplyTextString(helpText), nil)
return dispatcher.EndGroups
}

func silent(ctx *ext.Context, update *ext.Update) error {
user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
if err != nil {
logger.L.Errorf("Failed to obtain user: %s", err)
return dispatcher.EndGroups
}
if !user.Silent && user.DefaultStorage == "" {
ctx.Reply(update, ext.ReplyTextString("Please use /storage to set the default storage location first"), nil)
return dispatcher.EndGroups
}
user.Silent = !user.Silent
if err := dao.UpdateUser(user); err != nil {
logger.L.Errorf("Update user failed: %s", err)
ctx.Reply(update, ext.ReplyTextString("Update user failed"), nil)
return dispatcher.EndGroups
}
ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("%s silent mode", map[bool]string{true: "on", false: "off"}[user.Silent])), nil)
return dispatcher.EndGroups
}

func saveCmd(ctx *ext.Context, update *ext.Update) error {
res, ok := update.EffectiveMessage.GetReplyTo()
if !ok || res == nil {
ctx.Reply(update, ext.ReplyTextString("Please reply to the file you want to save"), nil)
return dispatcher.EndGroups
}
replyHeader, ok := res.(*tg.MessageReplyHeader)
if !ok {
ctx.Reply(update, ext.ReplyTextString("Please reply to the file you want to save"), nil)
return dispatcher.EndGroups
}
replyToMsgID, ok := replyHeader.GetReplyToMsgID()
if !ok {
ctx.Reply(update, ext.ReplyTextString("Please reply to the file you want to save"), nil)
return dispatcher.EndGroups
}

user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
if err != nil {
logger.L.Errorf("Failed to obtain user: %s", err)
ctx.Reply(update, ext.ReplyTextString("Failed to get user"), nil)
return dispatcher.EndGroups
}

storages := storage.GetUserStorages(user.ChatID)

if len(storages) == 0 {
ctx.Reply(update, ext.ReplyTextString("No storage available"), nil)
return dispatcher.EndGroups
}

msg, err := GetTGMessage(ctx, update.EffectiveChat().GetID(), replyToMsgID)
if err != nil {
logger.L.Errorf("Failed to get message: %s", err)
ctx.Reply(update, ext.ReplyTextString("Unable to get message"), nil)
return dispatcher.EndGroups
}

supported, _ := supportedMediaFilter(msg)
if !supported {
ctx.Reply(update, ext.ReplyTextString("Unsupported message type or no file in the message"), nil)
return dispatcher.EndGroups
}

replied, err := ctx.Reply(update, ext.ReplyTextString("Getting file information..."), nil)
if err != nil {
logger.L.Errorf("Reply failed: %s", err)
return dispatcher.EndGroups
}

cmdText := update.EffectiveMessage.Text
customFileName := strings.TrimSpace(strings.TrimPrefix(cmdText, "/save"))

file, err := FileFromMessage(ctx, update.EffectiveChat().GetID(), msg.ID, customFileName)
if err != nil {
logger.L.Errorf("Failed to get file: %s", err)
ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: fmt.Sprintf("Failed to get file: %s", err),
ID: replied.ID,
})
return dispatcher.EndGroups
}

// TODO: better file name
if file.FileName == "" {
file.FileName = fmt.Sprintf("%d_%d_%s", update.EffectiveChat().GetID(), replyToMsgID, file.Hash())
}
receivedFile := &types.ReceivedFile{
Processing: false,
FileName: file.FileName,
ChatID: update.EffectiveChat().GetID(),
MessageID: replyToMsgID,
ReplyMessageID: replied.ID,
ReplyChatID: update.GetUserChat().GetID(),
}

if err := dao.SaveReceivedFile(receivedFile); err != nil {
logger.L.Errorf("Failed to save received file: %s", err)
if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: fmt.Sprintf("Failed to save received file: %s", err),
ID: replied.ID,
}); err != nil {
logger.L.Errorf("Failed to edit message: %s", err)
}
return dispatcher.EndGroups
}
if !user.Silent || user.DefaultStorage == "" {
return ProvideSelectMessage(ctx, update, file, update.EffectiveChat().GetID(), msg.ID, replied.ID)
}
return HandleSilentAddTask(ctx, update, user, &types.Task{
Ctx: ctx,
Status: types.Pending,
File: file,
StorageName: user.DefaultStorage,
FileChatID: update.EffectiveChat().GetID(),
ReplyMessageID: replied.ID,
ReplyChatID: update.GetUserChat().GetID(),
FileMessageID: msg.ID,
UserID: user.ChatID,
})
}

func storageCmd(ctx *ext.Context, update *ext.Update) error {
user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
if err != nil {
logger.L.Errorf("Failed to obtain user: %s", err)
ctx.Reply(update, ext.ReplyTextString("Failed to get user"), nil)
return dispatcher.EndGroups
}
storages := storage.GetUserStorages(user.ChatID)
if len(storages) == 0 {
ctx.Reply(update, ext.ReplyTextString("No storage available"), nil)
return dispatcher.EndGroups
}

ctx.Reply(update, ext.ReplyTextString("Please select the storage location to be set as the default"), &ext.ReplyOpts{
Markup: getSetDefaultStorageMarkup(user.ChatID, storages),
})

return dispatcher.EndGroups
}

func setDefaultStorage(ctx *ext.Context, update *ext.Update) error {
args := strings.Split(string(update.CallbackQuery.Data), " ")
userID, _ := strconv.Atoi(args[1])
storageNameHash := args[2]
if userID != int(update.CallbackQuery.GetUserID()) {
ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
QueryID: update.CallbackQuery.QueryID,
Alert: true,
Message: "You do not have permission",
CacheTime: 5,
})
return dispatcher.EndGroups
}
storageName := storageHashName[storageNameHash]
selectedStorage, err := storage.GetStorageByName(storageName)

if err != nil {
logger.L.Errorf("Failed to obtain the specified storage: %s", err)
ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
QueryID: update.CallbackQuery.QueryID,
Alert: true,
Message: "Failed to obtain the specified storage",
CacheTime: 5,
})
return dispatcher.EndGroups
}
user, err := dao.GetUserByChatID(int64(userID))
if err != nil {
logger.L.Errorf("Failed to get user: %s", err)
ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
QueryID: update.CallbackQuery.QueryID,
Alert: true,
Message: "Failed to obtain user",
CacheTime: 5,
})
return dispatcher.EndGroups
}
user.DefaultStorage = storageName
if err := dao.UpdateUser(user); err != nil {
logger.L.Errorf("Failed to update user: %s", err)
ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
QueryID: update.CallbackQuery.QueryID,
Alert: true,
Message: "Update user failed",
CacheTime: 5,
})
return dispatcher.EndGroups
}
ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: fmt.Sprintf("%s (%s) has been set as the default storage location", selectedStorage.Name(), selectedStorage.Type()),
ID: update.CallbackQuery.GetMsgID(),
})
return dispatcher.EndGroups
}

func handleFileMessage(ctx *ext.Context, update *ext.Update) error {
logger.L.Trace("Got media: ", update.EffectiveMessage.Media.TypeName())
supported, err := supportedMediaFilter(update.EffectiveMessage.Message)
if err != nil {
return err
}
if !supported {
return dispatcher.EndGroups
}

user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
if err != nil {
logger.L.Errorf("Failed to obtain user: %s", err)
ctx.Reply(update, ext.ReplyTextString("Failed to get user"), nil)
return dispatcher.EndGroups
}
storages := storage.GetUserStorages(user.ChatID)
if len(storages) == 0 {
ctx.Reply(update, ext.ReplyTextString("No storage available"), nil)
return dispatcher.EndGroups
}

msg, err := ctx.Reply(update, ext.ReplyTextString("Getting file information..."), nil)
if err != nil {
logger.L.Errorf("Reply failed: %s", err)
return dispatcher.EndGroups
}
media := update.EffectiveMessage.Media
file, err := FileFromMedia(media, "")
if err != nil {
logger.L.Errorf("Failed to get file: %s", err)
ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("Failed to get file: %s", err)), nil)
return dispatcher.EndGroups
}
if file.FileName == "" {
file.FileName = fmt.Sprintf("%d_%d_%s", update.EffectiveChat().GetID(), update.EffectiveMessage.ID, file.Hash())
}

if err := dao.SaveReceivedFile(&types.ReceivedFile{
Processing: false,
FileName: file.FileName,
ChatID: update.EffectiveChat().GetID(),
MessageID: update.EffectiveMessage.ID,
ReplyMessageID: msg.ID,
ReplyChatID: update.GetUserChat().GetID(),
}); err != nil {
logger.L.Errorf("Failed to add received file: %s", err)
if _, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: fmt.Sprintf("Failed to add received file: %s", err),
ID: msg.ID,
}); err != nil {
logger.L.Errorf("Failed to edit message: %s", err)
}
return dispatcher.EndGroups
}

if !user.Silent || user.DefaultStorage == "" {
return ProvideSelectMessage(ctx, update, file, update.EffectiveChat().GetID(), update.EffectiveMessage.ID, msg.ID)
}
return HandleSilentAddTask(ctx, update, user, &types.Task{
Ctx: ctx,
Status: types.Pending,
File: file,
StorageName: user.DefaultStorage,
FileChatID: update.EffectiveChat().GetID(),
ReplyMessageID: msg.ID,
ReplyChatID: update.GetUserChat().GetID(),
FileMessageID: update.EffectiveMessage.ID,
UserID: user.ChatID,
})
}

func AddToQueue(ctx *ext.Context, update *ext.Update) error {
if !slice.Contain(config.Cfg.GetUsersID(), update.CallbackQuery.UserID) {
ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
QueryID: update.CallbackQuery.QueryID,
Alert: true,
Message: "You do not have permission",
CacheTime: 5,
})
return dispatcher.EndGroups
}
args := strings.Split(string(update.CallbackQuery.Data), " ")
fileChatID, _ := strconv.Atoi(args[1])
fileMessageID, _ := strconv.Atoi(args[2])
storageNameHash := args[3]
storageName := storageHashName[storageNameHash]
if storageName == "" {
logger.L.Errorf("Unknown storage location hash: %d", storageNameHash)
ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
QueryID: update.CallbackQuery.QueryID,
Alert: true,
Message: "Unknown storage location",
CacheTime: 5,
})
return dispatcher.EndGroups
}
logger.L.Tracef("Got add to queue: chatID: %d, messageID: %d, storage: %s", fileChatID, fileMessageID, storageName)
record, err := dao.GetReceivedFileByChatAndMessageID(int64(fileChatID), fileMessageID)
if err != nil {
logger.L.Errorf("Failed to obtain record: %s", err)
ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
QueryID: update.CallbackQuery.QueryID,
Alert: true,
Message: "Query record failed",
CacheTime: 5,
})
return dispatcher.EndGroups
}
if update.CallbackQuery.MsgID != record.ReplyMessageID {
record.ReplyMessageID = update.CallbackQuery.MsgID
if err := dao.SaveReceivedFile(record); err != nil {
logger.L.Errorf("Failed to update received file: %s", err)
}
}
file, err := FileFromMessage(ctx, record.ChatID, record.MessageID, record.FileName)
if err != nil {
logger.L.Errorf("Failed to get the file in the message: %s", err)
ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
QueryID: update.CallbackQuery.QueryID,
Alert: true,
Message: fmt.Sprintf("Failed to get the file in the message: %s", err),
CacheTime: 5,
})
return dispatcher.EndGroups
}

queue.AddTask(types.Task{
Ctx: ctx,
Status: types.Pending,
File: file,
StorageName: storageName,
FileChatID: record.ChatID,
ReplyMessageID: record.ReplyMessageID,
FileMessageID: record.MessageID,
ReplyChatID: record.ReplyChatID,
UserID: update.EffectiveUser().GetID(),
})

entityBuilder := entity.Builder{}
var entities []tg.MessageEntityClass
text := fmt.Sprintf("Added to the task queue\nFile name: %s\nCurrent number of queued tasks: %d", record.FileName, queue.Len())
if err := styling.Perform(&entityBuilder,
styling.Plain("Added to task queue\nFile name: "),
styling.Code(record.FileName),
styling.Plain("\nCurrent number of queued tasks: "),
styling.Bold(strconv.Itoa(queue.Len())),
); err != nil {
logger.L.Errorf("Failed to build entity: %s", err)
} else {
text, entities = entityBuilder.Complete()
}

ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: text,
Entities: entities,
ID: record.ReplyMessageID,
})
return dispatcher.EndGroups
}
