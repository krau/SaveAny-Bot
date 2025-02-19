package bot

import (
"errors"
"fmt"
"time"

"github.com/celestix/gotgproto/dispatcher"
"github.com/celestix/gotgproto/ext"
"github.com/gotd/td/telegram/message/entity"
"github.com/gotd/td/telegram/message/styling"
"github.com/gotd/td/tg"
"github.com/krau/SaveAny-Bot/common"
"github.com/krau/SaveAny-Bot/dao"
"github.com/krau/SaveAny-Bot/logger"
"github.com/krau/SaveAny-Bot/queue"
"github.com/krau/SaveAny-Bot/storage"
"github.com/krau/SaveAny-Bot/types"
)

var (
ErrEmptyDocument = errors.New("document is empty")
ErrEmptyPhoto = errors.New("photo is empty")
ErrEmptyPhotoSize = errors.New("photo size is empty")
ErrEmptyPhotoSizes = errors.New("photo size slice is empty")
ErrNoStorages = errors.New("no available storage")
)

func supportedMediaFilter(m *tg.Message) (bool, error) {
if not := m.Media == nil; not {
return false, dispatcher.EndGroups
}
switch m.Media.(type) {
case *tg.MessageMediaDocument:
return true, nil
case *tg.MessageMediaPhoto:
return true, nil
default:
return false, nil
}
}

// for callback data
var storageHashName = map[string]string{}

func getSelectStorageMarkup(userChatID int64, fileChatID, fileMessageID int) (*tg.ReplyInlineMarkup, error) {
user, err := dao.GetUserByChatID(userChatID)
if err != nil {
return nil, err
}
storages := storage.GetUserStorages(user.ChatID)

buttons := make([]tg.KeyboardButtonClass, 0)
for _, storage := range storages {
nameHash := common.HashString(storage.Name())
storageHashName[nameHash] = storage.Name()
buttons = append(buttons, &tg.KeyboardButtonCallback{
Text: storage.Name(),
Data: []byte(fmt.Sprintf("add %d %d %s", fileChatID, fileMessageID, nameHash)),
})
}
markup := &tg.ReplyInlineMarkup{}
for i := 0; i < len(buttons); i += 3 {
row := tg.KeyboardButtonRow{}
row.Buttons = buttons[i:min(i+3, len(buttons))]
markup.Rows = append(markup.Rows, row)
}
return markup, nil
}

func getSetDefaultStorageMarkup(userChatID int64, storages []storage.Storage) *tg.ReplyInlineMarkup {
buttons := make([]tg.KeyboardButtonClass, 0)
for _, storage := range storages {
nameHash := common.HashString(storage.Name())
storageHashName[nameHash] = storage.Name()
buttons = append(buttons, &tg.KeyboardButtonCallback{
Text: storage.Name(),
Data: []byte(fmt.Sprintf("set_default %d %s", userChatID, nameHash)),
})
}
markup := &tg.ReplyInlineMarkup{}
for i := 0; i < len(buttons); i += 3 {
row := tg.KeyboardButtonRow{}
row.Buttons = buttons[i:min(i+3, len(buttons))]
markup.Rows = append(markup.Rows, row)
}
return markup

}

func FileFromMedia(media tg.MessageMediaClass, customFileName string) (*types.File, error) {
switch media := media.(type) {
case *tg.MessageMediaDocument:
document, ok := media.Document.AsNotEmpty()
if !ok {
return nil, ErrEmptyDocument
}
if customFileName != "" {
return &types.File{
Location: document.AsInputDocumentFileLocation(),
FileSize: document.Size,
FileName: customFileName,
}, nil
}
fileName := ""
for _, attribute := range document.Attributes {
if name, ok := attribute.(*tg.DocumentAttributeFilename); ok {
fileName = name.GetFileName()
break
}
}
return &types.File{
Location: document.AsInputDocumentFileLocation(),
FileSize: document.Size,
FileName: fileName,
}, nil
case *tg.MessageMediaPhoto:
photo, ok := media.Photo.AsNotEmpty()
if !ok {
return nil, ErrEmptyPhoto
}
sizes := photo.Sizes
if len(sizes) == 0 {
return nil, ErrEmptyPhotoSizes
}
photoSize := sizes[len(sizes)-1]
size, ok := photoSize.AsNotEmpty()
if !ok {
return nil, ErrEmptyPhotoSize
}
location := new(tg.InputPhotoFileLocation)
location.ID = photo.GetID()
location.AccessHash = photo.GetAccessHash()
location.FileReference = photo.GetFileReference()
location.ThumbSize = size.GetType()
fileName := customFileName
if fileName == "" {
fileName = fmt.Sprintf("photo_%s_%d.jpg", time.Now().Format("2006-01-02_15-04-05"), photo.GetID())
}
return &types.File{
Location: location,
FileSize: 0,
FileName: fileName,
}, nil

}
return nil, fmt.Errorf("unexpected type %T", media)
}

func FileFromMessage(ctx *ext.Context, chatID int64, messageID int, customFileName string) (*types.File, error) {
key := fmt.Sprintf("file:%d:%d", chatID, messageID)
logger.L.Debugf("Getting file: %s", key)
var cachedFile types.File
err := common.Cache.Get(key, &cachedFile)
if err == nil {
return &cachedFile, nil
}
message, err := GetTGMessage(ctx, chatID, messageID)
if err != nil {
return nil, err
}
file, err := FileFromMedia(message.Media, customFileName)
if err != nil {
return nil, err
}
if err := common.Cache.Set(key, file, 3600); err != nil {
logger.L.Errorf("Failed to cache file: %s", err)
}
return file, nil
}

func GetTGMessage(ctx *ext.Context, chatId int64, messageID int) (*tg.Message, error) {
logger.L.Debugf("Fetching message: %d", messageID)
messages, err := ctx.GetMessages(chatId, []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}})
if err != nil {
return nil, err
}
if len(messages) == 0 {
return nil, errors.New("no messages found")
}
msg := messages[0]
tgMessage, ok := msg.(*tg.Message)
if !ok {
return nil, fmt.Errorf("unexpected message type: %T", msg)
}
return tgMessage, nil
}

func ProvideSelectMessage(ctx *ext.Context, update *ext.Update, file *types.File, chatID int64, fileMsgID, toEditMsgID int) error {
entityBuilder := entity.Builder{}
var entities []tg.MessageEntityClass
text := fmt.Sprintf("File name: %s\nPlease select the storage location", file.FileName)
if err := styling.Perform(&entityBuilder,
styling.Plain("File name: "),
styling.Code(file.FileName),
styling.Plain("\nPlease select the storage location"),
); err != nil {
logger.L.Errorf("Failed to build entity: %s", err)
} else {
text, entities = entityBuilder.Complete()
}
markup, err := getSelectStorageMarkup(update.EffectiveUser().GetID(), int(chatID), fileMsgID)
if errors.Is(err, ErrNoStorages) {
logger.L.Errorf("Failed to get select storage markup: %s", err)
ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: "No storage available",
ID: toEditMsgID,
})
return dispatcher.EndGroups
} else if err != nil {
logger.L.Errorf("Failed to get select storage markup: %s", err)
ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: "Unable to obtain storage",
ID: toEditMsgID,
})
return dispatcher.EndGroups
}
_, err = ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: text,
Entities: entities,
ReplyMarkup: markup,
ID: toEditMsgID,
})
if err != nil {
logger.L.Errorf("Failed to reply: %s", err)
}
return dispatcher.EndGroups
}

func HandleSilentAddTask(ctx *ext.Context, update *ext.Update, user *types.User, task *types.Task) error {
if user.DefaultStorage == "" {
ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: "Please use /storage to set the default storage location first",
ID: task.ReplyMessageID,
})
return dispatcher.EndGroups
}
queue.AddTask(*task)
ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
Message: fmt.Sprintf("Added to queue: %s\nCurrent number of queued tasks: %d", task.FileName(), queue.Len()),
ID: task.ReplyMessageID,
})
return dispatcher.EndGroups
}
