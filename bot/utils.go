package bot

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/storage"
)

func supportedMediaFilter(m *types.Message) (bool, error) {
	if not := m.Media == nil; not {
		return false, dispatcher.EndGroups
	}
	switch m.Media.(type) {
	case *tg.MessageMediaDocument:
		return true, nil
	case *tg.MessageMediaWebPage:
		return false, dispatcher.EndGroups
	case tg.MessageMediaClass:
		return false, dispatcher.EndGroups
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
