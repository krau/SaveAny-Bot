(chatId, []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}})
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
