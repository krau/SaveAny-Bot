package bot

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
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

func getAddTaskMarkup(messageID int) *tg.ReplyInlineMarkup {
	storageButtons := make([]tg.KeyboardButtonClass, 0)
	for _, name := range storage.StorageKeys {
		storageButtons = append(storageButtons, &tg.KeyboardButtonCallback{
			Text: StorageDisplayNames[string(name)],
			Data: []byte(fmt.Sprintf("add %d %s", messageID, name)),
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
						Data: []byte(fmt.Sprintf("add %d all", messageID)),
					},
				},
			},
		},
	}
}

func FileFromMedia(media tg.MessageMediaClass) (*types.File, error) {
	switch media := media.(type) {
	case *tg.MessageMediaDocument:
		document, ok := media.Document.AsNotEmpty()
		if !ok {
			return nil, fmt.Errorf("document is empty")
		}
		var fileName string
		for _, attribute := range document.Attributes {
			if name, ok := attribute.(*tg.DocumentAttributeFilename); ok {
				fileName = name.GetFileName()
				break
			}
		}
		if fileName == "" {
			fileName = fmt.Sprintf("%x", md5.Sum(document.GetFileReference()))
			logger.L.Warnf("File name is empty, using hash: %s", fileName)
		}
		return &types.File{
			Location: document.AsInputDocumentFileLocation(),
			FileSize: document.Size,
			FileName: fileName,
		}, nil
	case *tg.MessageMediaPhoto:
		photo, ok := media.Photo.AsNotEmpty()
		if !ok {
			return nil, fmt.Errorf("photo is empty")
		}
		sizes := photo.Sizes
		if len(sizes) == 0 {
			return nil, fmt.Errorf("photo sizes is empty")
		}
		photoSize := sizes[len(sizes)-1]
		size, ok := photoSize.AsNotEmpty()
		if !ok {
			return nil, fmt.Errorf("photo size is empty")
		}
		location := new(tg.InputPhotoFileLocation)
		location.ID = photo.GetID()
		location.AccessHash = photo.GetAccessHash()
		location.FileReference = photo.GetFileReference()
		location.ThumbSize = size.GetType()
		return &types.File{
			Location: location,
			FileSize: 0,
			FileName: fmt.Sprintf("photo_%s_%d.jpg", time.Now().Format("2006-01-02_15-04-05"), photo.GetID()),
		}, nil

	}
	return nil, fmt.Errorf("unexpected type %T", media)
}

func FileFromMessage(ctx context.Context, client *gotgproto.Client, chatID int64, messageID int) (*types.File, error) {
	key := fmt.Sprintf("file:%d:%d", chatID, messageID)
	logger.L.Debugf("Getting file: %s", key)
	var cachedFile types.File
	err := common.Cache.Get(key, &cachedFile)
	if err == nil {
		return &cachedFile, nil
	}

	message, err := GetTGMessage(ctx, client, messageID)
	if err != nil {
		return nil, err
	}
	file, err := FileFromMedia(message.Media)
	if err != nil {
		return nil, err
	}
	if err := common.Cache.Set(key, file, 3600); err != nil {
		logger.L.Errorf("Failed to cache file: %s", err)
	}
	return file, nil
}

func GetTGMessage(ctx context.Context, client *gotgproto.Client, messageID int) (*tg.Message, error) {
	logger.L.Debugf("Fetching message: %d", messageID)
	res, err := client.API().MessagesGetMessages(ctx, []tg.InputMessageClass{
		&tg.InputMessageID{
			ID: messageID,
		},
	})
	if err != nil {
		return nil, err
	}
	messages := res.(*tg.MessagesMessages)
	msg := messages.Messages[0]
	if _, ok := msg.(*tg.Message); !ok {
		return nil, fmt.Errorf("unexpected type %T, this file may be deleted", msg)
	}
	return msg.(*tg.Message), nil
}
