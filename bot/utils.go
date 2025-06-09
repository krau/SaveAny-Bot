package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

var (
	ErrEmptyDocument   = errors.New("document is empty")
	ErrEmptyPhoto      = errors.New("photo is empty")
	ErrEmptyPhotoSize  = errors.New("photo size is empty")
	ErrEmptyPhotoSizes = errors.New("photo size slice is empty")
	ErrNoStorages      = errors.New("no available storage")
	ErrEmptyMessage    = errors.New("message is empty")
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

func getSelectStorageMarkup(userChatID int64, fileChatID, fileMessageID int) (*tg.ReplyInlineMarkup, error) {
	user, err := dao.GetUserByChatID(userChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by chat ID: %d, error: %w", userChatID, err)
	}
	storages := storage.GetUserStorages(user.ChatID)
	// if len(storages) == 0 {
	// 	return nil, ErrNoStorages
	// }

	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, storage := range storages {
		cbData := fmt.Sprintf("%d %d %s 0", fileChatID, fileMessageID, storage.Name()) // 0 for empty dir id
		cbDataId, err := dao.CreateCallbackData(cbData)
		if err != nil {
			return nil, fmt.Errorf("failed to create callback data: %w", err)
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: storage.Name(),
			Data: fmt.Appendf(nil, "add %d", cbDataId),
		})
	}
	markup := &tg.ReplyInlineMarkup{}
	for i := 0; i < len(buttons); i += 3 {
		row := tg.KeyboardButtonRow{}
		row.Buttons = buttons[i:min(i+3, len(buttons))]
		markup.Rows = append(markup.Rows, row)
	}
	markup.Rows = append(markup.Rows, tg.KeyboardButtonRow{
		Buttons: []tg.KeyboardButtonClass{
			&tg.KeyboardButtonCallback{
				Text: "发送到当前聊天",
				Data: []byte(fmt.Sprintf("send_here %d %d", fileChatID, fileMessageID)),
			},
		},
	})
	return markup, nil
}

func getSelectDirMarkup(fileChatID, fileMessageID int, storageName string, dirs []dao.Dir) (*tg.ReplyInlineMarkup, error) {
	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, dir := range dirs {
		if dir.ID == 0 || dir.StorageName != storageName {
			return nil, fmt.Errorf("unexpected dir: %v", dir)
		}
		cbDataId, err := dao.CreateCallbackData(fmt.Sprintf("%d %d %s %d", fileChatID, fileMessageID, storageName, dir.ID))
		if err != nil {
			return nil, fmt.Errorf("failed to create callback data: %w", err)
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: dir.Path,
			Data: []byte(fmt.Sprintf("add_to_dir %d", cbDataId)),
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

func getSetDefaultStorageMarkup(userChatID int64, storages []storage.Storage) (*tg.ReplyInlineMarkup, error) {
	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, storage := range storages {
		cbDataId, err := dao.CreateCallbackData(storage.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to create callback data: %w", err)
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: storage.Name(),
			Data: []byte(fmt.Sprintf("set_default %d %d", userChatID, cbDataId)),
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
	cachedFile, err := common.CacheGet[*types.File](ctx, key)
	if err == nil {
		if customFileName != "" {
			cachedFile.FileName = customFileName
		}
		return cachedFile, nil
	}
	common.Log.Debugf("Getting file: %s", key)
	message, err := GetTGMessage(ctx, chatID, messageID)
	if err != nil {
		return nil, err
	}
	file, err := FileFromMedia(message.Media, customFileName)
	if err != nil {
		return nil, err
	}
	if err := common.CacheSet(ctx, key, file); err != nil {
		common.Log.Errorf("Failed to cache file: %s", err)
	}
	return file, nil
}

func GetTGMessage(ctx *ext.Context, chatId int64, messageID int) (*tg.Message, error) {
	key := fmt.Sprintf("message:%d:%d", chatId, messageID)
	cacheMessage, err := common.CacheGet[*tg.Message](ctx, key)
	if err == nil {
		return cacheMessage, nil
	}
	common.Log.Debugf("Fetching message: %d:%d", chatId, messageID)
	messages, err := ctx.GetMessages(chatId, []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}})
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, ErrEmptyMessage
	}
	msg := messages[0]
	tgMessage, ok := msg.(*tg.Message)
	if !ok {
		return nil, fmt.Errorf("unexpected message type: %T", msg)
	}
	if err := common.CacheSet(ctx, key, tgMessage); err != nil {
		common.Log.Errorf("Failed to cache message: %s", err)
	}
	return tgMessage, nil
}

// Userbot only
//
// https://github.com/iyear/tdl/blob/fbb396da774ba544e527c3ef41c44921ad74ee98/core/util/tutil/tutil.go#L174
func GetSingleHistoryMessage(ctx context.Context, client *tg.Client, peer tg.InputPeerClass, msg int) (*tg.Message, error) {
	it := query.Messages(client).GetHistory(peer).OffsetID(msg + 1).BatchSize(1).Iter()

	if !it.Next(ctx) {
		return nil, fmt.Errorf("failed to get message %d from %s: %w", msg, peer, it.Err())
	}

	m, ok := it.Value().Msg.(*tg.Message)
	if !ok {
		return nil, fmt.Errorf("invalid message %d", msg)
	}

	if m.GetID() != msg {
		return nil, fmt.Errorf("the message %d/%d may be deleted", GetInputPeerID(peer), msg)
	}
	return m, nil
}

// Userbot only
func GetHistoryMessages(ctx context.Context, client *tg.Client, peer tg.InputPeerClass, startID, limit int) ([]*tg.Message, error) {
	endID := startID + limit - 1
	msgs, err := client.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:      peer,
		OffsetID:  startID,
		Limit:     limit,
		AddOffset: startID - endID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get history messages: %w", err)
	}
	var msgClass []tg.MessageClass
	switch msgsv := msgs.(type) {
	case *tg.MessagesMessages:
		msgClass = msgsv.GetMessages()
	case *tg.MessagesMessagesSlice:
		msgClass = msgsv.GetMessages()
	case *tg.MessagesChannelMessages:
		msgClass = msgsv.GetMessages()
	default:
		return nil, fmt.Errorf("unexpected messages type: %T", msgs)
	}

	messageBatch := make([]*tg.Message, 0, 100)

	for _, msg := range msgClass {
		msgNotEmpty, ok := msg.AsNotEmpty()
		if !ok {
			continue
		}
		switch msgNotEmptyV := msgNotEmpty.(type) {
		case *tg.Message:
			messageBatch = append(messageBatch, msgNotEmptyV)
		default:
			common.Log.Warnf("Unexpected message type: %T, skipping", msgNotEmptyV)
			continue
		}
	}
	if len(messageBatch) == 0 {
		return nil, fmt.Errorf("no messages found for peer %s with startID %d and limit %d", peer, startID, limit)
	}
	return messageBatch, nil
}

func GetMediaGroup(ctx *ext.Context, chatID int64, messageID int, groupID int64) ([]*tg.Message, error) {
	peer := ctx.PeerStorage.GetInputPeerById(chatID)
	if peer == nil {
		return nil, fmt.Errorf("无法获取聊天 %d 的输入 Peer", chatID)
	}
	messages, err := GetHistoryMessages(ctx, ctx.Raw, peer, messageID-9, 20)
	if err != nil {
		return nil, fmt.Errorf("获取消息失败: %s", err)
	}
	var groupMessages []*tg.Message
	for _, msg := range messages {
		gID, isGroup := msg.GetGroupedID()
		if isGroup && gID == groupID {
			groupMessages = append(groupMessages, msg)
		}
	}
	if len(groupMessages) == 0 || (len(groupMessages) == 1 && groupMessages[0].ID == messageID) {
		return nil, fmt.Errorf("未找到媒体组 %d 中的消息", groupID)
	}
	return groupMessages, nil
}

func GetInputPeerID(peer tg.InputPeerClass) int64 {
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		return p.UserID
	case *tg.InputPeerChat:
		return p.ChatID
	case *tg.InputPeerChannel:
		return p.ChannelID
	}

	return 0
}

func ProvideSelectMessage(ctx *ext.Context, update *ext.Update, fileName string, chatID int64, fileMsgID, toEditMsgID int) error {
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	text := fmt.Sprintf("文件名: %s\n请选择存储位置", fileName)
	if err := styling.Perform(&entityBuilder,
		styling.Plain("文件名: "),
		styling.Code(fileName),
		styling.Plain("\n请选择存储位置"),
	); err != nil {
		common.Log.Errorf("Failed to build entity: %s", err)
	} else {
		text, entities = entityBuilder.Complete()
	}
	markup, err := getSelectStorageMarkup(update.GetUserChat().GetID(), int(chatID), fileMsgID)
	if errors.Is(err, ErrNoStorages) {
		common.Log.Errorf("Failed to get select storage markup: %s", err)
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: "无可用存储",
			ID:      toEditMsgID,
		})
		return dispatcher.EndGroups
	} else if err != nil {
		common.Log.Errorf("Failed to get select storage markup: %s", err)
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: "无法获取存储",
			ID:      toEditMsgID,
		})
		return dispatcher.EndGroups
	}
	_, err = ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		Message:     text,
		Entities:    entities,
		ReplyMarkup: markup,
		ID:          toEditMsgID,
	})
	if err != nil {
		common.Log.Errorf("Failed to reply: %s", err)
	}
	return dispatcher.EndGroups
}

func HandleSilentAddTask(ctx *ext.Context, update *ext.Update, user *dao.User, task *types.Task) error {
	if user.DefaultStorage == "" {
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: "请先使用 /storage 设置默认存储位置",
			ID:      task.ReplyMessageID,
		})
		return dispatcher.EndGroups
	}
	queue.AddTask(task)
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("已添加到队列: %s\n当前排队任务数: %d", task.FileName(), queue.Len()),
		ID:      task.ReplyMessageID,
	})
	return dispatcher.EndGroups
}

func GenFileNameFromMessage(message tg.Message, file *types.File) string {
	if file.FileName != "" {
		return file.FileName
	}
	fileName := genFileNameFromMessageText(message, file)
	media, ok := message.GetMedia()
	if !ok {
		return fileName
	}
	ext, ok := extraMediaExt(media)
	if ok {
		return fileName + ext
	}
	return fileName
}

func genFileNameFromMessageText(message tg.Message, file *types.File) string {
	text := strings.TrimSpace(message.GetMessage())
	if text == "" {
		return file.Hash()
	}
	tags := common.ExtractTagsFromText(text)
	if len(tags) > 0 {
		return fmt.Sprintf("%s_%s", strings.Join(tags, "_"), strconv.Itoa(message.GetID()))
	}
	// 删除换行和特殊字符
	text = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			return '_'
		}
		return r
	}, text)
	runes := []rune(text)
	return string(runes[:min(128, len(runes))])
}

func extraMediaExt(media tg.MessageMediaClass) (string, bool) {
	switch media := media.(type) {
	case *tg.MessageMediaDocument:
		doc, ok := media.Document.AsNotEmpty()
		if !ok {
			return "", false
		}
		ext := mimetype.Lookup(doc.MimeType).Extension()
		if ext == "" {
			return "", false
		}
		return ext, true
	case *tg.MessageMediaPhoto:
		return ".jpg", true
	}
	return "", false
}
