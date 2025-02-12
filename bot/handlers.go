package bot

import (
"errors"
"fmt"
"strconv"
"strings"

"github.com/duke-git/lancet/v2/slice"
"github.com/gookit/goutil/maputil"
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
dispatcher.AddHandler(handlers.NewCommand("storage", setDefaultStorage))
dispatcher.AddHandler(handlers.NewCommand("save", saveCmd))
dispatcher.AddHandler(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("add"), AddToQueue))
dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, handleFileMessage))
}

const noPermissionText string = `
This Bot is for personal use only.
You can deploy your own instance: https://github.com/krau/SaveAny-Bot
`

func checkPermission(ctx *ext.Context, update *ext.Update) error {
userID := update.GetUserChat().GetID()
if !slice.Contain(config.Cfg.Telegram.Admins, userID) {
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
SaveAny Bot - Save your Telegram files
Order:
/start - Start using
/help - Display help
/silent - silent mode
/storage - Sets the default storage location
/save [custom file name] - save the file

Silent mode: When enabled, Bot saves received files directly to the default location without asking again
`

func help(ctx *ext.Context, update *ext.Update) error {
ctx.Reply(update, ext.ReplyTextString(helpText), nil)
return dispatcher.EndGroups
}

func silent(ctx *ext.Context, update *ext.Update) error {
user, err := dao.GetUserByUserID(update.GetUserChat().GetID())
if err != nil {
logger.L.Errorf("Failed to get user: %s", err)
return dispatcher.EndGroups
}
user.Silent = !user.Silent
if err := dao.UpdateUser(user); err != nil {
logger.L.Errorf("Failed to update user: %s", err)
return dispatcher.EndGroups
}
ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("%s silent mode", func() string {
if user.Silent {
return "on"
}
return "Close"
}())), nil)
return dispatcher.EndGroups
}

func setDefaultStorage(ctx *ext.Context, update *ext.Update) error {
if len(storage.Storages) == 0 {
ctx.Reply(update, ext.ReplyTextString("Storage not configured"), nil)
return dispatcher.EndGroups
}
args := strings.Split(update.EffectiveMessage.Text, " ")
avaliableStorages := maputil.Keys(storage.Storages)
if len(args) < 2 {
text := []styling.StyledTextOption{
styling.Plain("Please provide a storage location name, available items:"),
}
for _, name := range avaliableStorages {
text = append(text, styling.Plain("\n"))
text = append(text, styling.Code(name))
}
text = append(text, styling.Plain("\nExample: /storage local"))
ctx.Reply(update, ext.ReplyTextStyledTextArray(text), nil)
return dispatcher.EndGroups
}
storageName := args[1]
if !slice.Contain(avaliableStorages, storageName) {
ctx.Reply(update, ext.ReplyTextString("The storage location does not exist"), nil)
return dispatcher.EndGroups
}
user, err := dao.GetUserByUserID(update.GetUserChat().GetID())
if err != nil {
logger.L.Errorf("Failed to get user: %s", err)
return dispatcher.EndGroups
}
user.DefaultStorage = storageName
if err := dao.UpdateUser(user); err != nil {
logger.L.Errorf("Failed to update user: %s", err)
return dispatcher.EndGroups
}
ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("The default storage location has been set to %s", storageName)), nil)
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

msg, err := GetTGMessage(ctx, Client, replyToMsgID)

supported, _ := supportedMediaFilter(msg)
if !supported {
ctx.Reply(update, ext.ReplyTextString("Unsupported message type or no file in the message"), nil)
return dispatcher.EndGroups
}

user, err := dao.GetUserByUserID(update.GetUserChat().GetID())
if err != nil {
logger.L.Errorf("Failed to get user: %s", err)
return dispatcher.EndGroups
}

replied, err := ctx.Reply(update, ext.ReplyTextString("Getting file information..."), nil)
if err != nil {
logger.L.Errorf("Failed to reply: %s", err)
return dispatcher.EndGroups
}

cmdText := update.EffectiveMessage.Text
customFileName := strings.TrimSpace(strings.TrimPrefix(cmdText, "/save"))

file, err := FileFromMessage(ctx, Client, update.EffectiveChat().GetID(), msg.ID, customFileName)
if err != nil {
logger.L.Errorf("Failed to get file from message: %s", err)
ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: "Unable to get file",
ID: replied.ID,
})
return dispatcher.EndGroups
}

if file.FileName == "" {
ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: "Unable to get file name",
ID: replied.ID,
})
return dispatcher.EndGroups
}

receivedFile := &types.ReceivedFile{
Processing: false,
FileName: file.FileName,
ChatID: update.EffectiveChat().GetID(),
MessageID: replyToMsgID,
ReplyMessageID: replied.ID,
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

if !user.Silent {
text := "Please select a storage location"
_, err = ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: text,
ReplyMarkup: getAddTaskMarkup(msg.ID),
ID: replied.ID,
})
if err != nil {
logger.L.Errorf("Failed to reply: %s", err)
}
return dispatcher.EndGroups
}

if user.DefaultStorage == "" {
ctx.Reply(update, ext.ReplyTextString("Please use /storage to set the default storage location first"), nil)
return dispatcher.EndGroups
}
queue.AddTask(types.Task{
Ctx: ctx,
Status: types.Pending,
File: file,
Storage: types.StorageType(user.DefaultStorage),
ChatID: update.EffectiveChat().GetID(),
ReplyMessageID: replied.ID,
MessageID: msg.ID,
})
_, err = ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: fmt.Sprintf("Added to queue: %s\nCurrent number of queued tasks: %d", file.FileName, queue.Len()),
ID: replied.ID,
})
if err != nil {
logger.L.Errorf("Failed to edit message: %s", err)
}
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

user, err := dao.GetUserByUserID(update.GetUserChat().GetID())
if err != nil {
logger.L.Errorf("Failed to get user: %s", err)
return dispatcher.EndGroups
}

msg, err := ctx.Reply(update, ext.ReplyTextString("Getting file information..."), nil)
if err != nil {
logger.L.Errorf("Failed to reply: %s", err)
return dispatcher.EndGroups
}
media := update.EffectiveMessage.Media
file, err := FileFromMedia(media, "")
if err != nil {
logger.L.Errorf("Failed to get file from media: %s", err)
if errors.Is(err, ErrEmptyFileName) {
ctx.Reply(update, ext.ReplyTextString("Unable to get the file name, please use /save <custom file name> to reply to this file"), nil)
} else {
ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("Failed to get file: %s", err)), nil)
}
return dispatcher.EndGroups
}
if file.FileName == "" {
ctx.Reply(update, ext.ReplyTextString("Unable to get file name"), nil)
return dispatcher.EndGroups
}

if err := dao.SaveReceivedFile(&types.ReceivedFile{
Processing: false,
FileName: file.FileName,
ChatID: update.EffectiveChat().GetID(),
MessageID: update.EffectiveMessage.ID,
ReplyMessageID: msg.ID,
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

if !user.Silent {
text := "Please select a storage location"
_, err = ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: text,
ReplyMarkup: getAddTaskMarkup(update.EffectiveMessage.ID),
ID: msg.ID,
})
if err != nil {
logger.L.Errorf("Failed to edit message: %s", err)
}
return dispatcher.EndGroups
}

if user.DefaultStorage == "" {
ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: "Please use /storage to set the default storage location first",
ID: msg.ID,
})
return dispatcher.EndGroups
}

queue.AddTask(types.Task{
Ctx: ctx,
Status: types.Pending,
File: file,
Storage: types.StorageType(user.DefaultStorage),
ChatID:update.EffectiveChat().GetID(),
ReplyMessageID: msg.ID,
MessageID: update.EffectiveMessage.ID,
})

ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: fmt.Sprintf("Added to queue: %s\nCurrent number of queued tasks: %d", file.FileName, queue.Len()),
ID: msg.ID,
})
return dispatcher.EndGroups
}

func AddToQueue(ctx *ext.Context, update *ext.Update) error {
if !slice.Contain(config.Cfg.Telegram.Admins, update.CallbackQuery.UserID) {
ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
QueryID: update.CallbackQuery.QueryID,
Alert: true,
Message: "You do not have permission",
CacheTime: 5,
})
return dispatcher.EndGroups
}
args := strings.Split(string(update.CallbackQuery.Data), " ")
messageID, _ := strconv.Atoi(args[1])
logger.L.Tracef("Got add to queue: chatID: %d, messageID: %d, storage: %s", update.EffectiveChat().GetID(), messageID, args[2])
record, err := dao.GetReceivedFileByChatAndMessageID(update.EffectiveChat().GetID(), messageID)
if err != nil {
logger.L.Errorf("Failed to get received file: %s", err)
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

file, err := FileFromMessage(ctx, Client, record.ChatID, record.MessageID, record.FileName)
if err != nil {
logger.L.Errorf("Failed to get file from message: %s", err)
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
Storage: types.StorageType(args[2]),
ChatID: record.ChatID,
ReplyMessageID: record.ReplyMessageID,
MessageID: record.MessageID,
})
ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: fmt.Sprintf("Added to queue: %s\nCurrent number of queued tasks: %d", record.FileName, queue.Len()),
ID: record.ReplyMessageID,
})
return dispatcher.EndGroups
}
