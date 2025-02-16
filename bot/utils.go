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
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/queue"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

var (
	ErrEmptyDocument   = errors.New("document is empty")
	ErrEmptyPhoto      = errors.New("photo is empty")
	ErrEmptyPhotoSize  = errors.New("photo size is empty")
	ErrEmptyPhotoSizes = errors.New("photo size slice is empty")
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

var StorageDisplayNames = map[string]string{
	"all":    "全部",
	"local":  "服务器磁盘",
	"alist":  "Alist",
	"webdav": "WebDAV",
}

func getAddTaskMarkup(chatID, messageID int) *tg.ReplyInlineMarkup {
	storageButtons := make([]tg.KeyboardButtonClass, 0)
	for _, name := range storage.StorageKeys {
		storageButtons = append(storageButtons, &tg.KeyboardButtonCallback{
			Text: StorageDisplayNames[string(name)],
			Data: []byte(fmt.Sprintf("add %d %d %s", chatID, messageID, name)),
		})
	}

	if len(storageButtons) < 1 {
		return nil
	}
	if len(storageButtons) == 1 {
		return &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: storageButtons,
				},
			},
		}
	}
	return &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: storageButtons,
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "全部",
						Data: []byte(fmt.Sprintf("add %d %d all", chatID, messageID)),
					},
				},
			},
		},
	}
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

func ProvideSelectMessage(ctx *ext.Context, update *ext.Update, file *types.File, chatID int, fileMsgID, toEditMsgID int) error {
	entityBuilder := entity.Builder{}
	var entities []tg.MessageEntityClass
	text := fmt.Sprintf("文件名: %s\n请选择存储位置", file.FileName)
	if err := styling.Perform(&entityBuilder,
		styling.Plain("文件名: "),
		styling.Code(file.FileName),
		styling.Plain("\n请选择存储位置"),
	); err != nil {
		logger.L.Errorf("Failed to build entity: %s", err)
	} else {
		text, entities = entityBuilder.Complete()
	}
	_, err := ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		Message:     text,
		Entities:    entities,
		ReplyMarkup: getAddTaskMarkup(chatID, fileMsgID),
		ID:          toEditMsgID,
	})
	if err != nil {
		logger.L.Errorf("Failed to reply: %s", err)
	}
	return dispatcher.EndGroups
}

func HandleSilentAddTask(ctx *ext.Context, update *ext.Update, user *types.User, task *types.Task) error {
	if user.DefaultStorage == "" {
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: "请先使用 /storage 设置默认存储位置",
			ID:      task.ReplyMessageID,
		})
		return dispatcher.EndGroups
	}
	queue.AddTask(*task)
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		Message: fmt.Sprintf("已添加到队列: %s\n当前排队任务数: %d", task.FileName(), queue.Len()),
		ID:      task.ReplyMessageID,
	})
	return dispatcher.EndGroups
}
