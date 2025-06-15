package msgelem

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func BuildAddOneSelectStorageKeyboard(stors []storage.Storage, file tfile.TGFileMessage) (*tg.ReplyInlineMarkup, error) {
	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, storage := range stors {
		data := tcbdata.Add{
			Files:            []tfile.TGFileMessage{file},
			SelectedStorName: storage.Name(),
			AsBatch:          false,
		}
		dataid := xid.New().String()
		err := cache.Set(dataid, data)
		if err != nil {
			return nil, err
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: storage.Name(),
			Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dataid),
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

func BuildAddOneSelectStorageMessage(ctx context.Context, stors []storage.Storage, file tfile.TGFileMessage, msgId int) (*tg.MessagesEditMessageRequest, error) {
	eb := entity.Builder{}
	var entities []tg.MessageEntityClass
	text := fmt.Sprintf("文件名: %s\n请选择存储位置", file.Name())
	if err := styling.Perform(&eb,
		styling.Plain("文件名: "),
		styling.Code(file.Name()),
		styling.Plain("\n请选择存储位置"),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entity: %s", err)
	} else {
		text, entities = eb.Complete()
	}
	markup, err := BuildAddOneSelectStorageKeyboard(stors, file)
	if err != nil {
		return nil, fmt.Errorf("failed to build storage keyboard: %w", err)
	}
	return &tg.MessagesEditMessageRequest{
		Message:     text,
		Entities:    entities,
		ReplyMarkup: markup,
		ID:          msgId,
	}, nil
}

func BuildSetDefaultStorageMarkup(ctx context.Context, userID int64, stors []storage.Storage) (*tg.ReplyInlineMarkup, error) {
	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, storage := range stors {
		data := tcbdata.SetDefaultStorage{
			StorageName: storage.Name(),
		}
		dataid := xid.New().String()
		err := cache.Set(dataid, data)
		if err != nil {
			return nil, err
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: storage.Name(),
			Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeSetDefault, dataid),
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

func BuildAddBatchSelectStorageKeyboard(stors []storage.Storage, files []tfile.TGFileMessage) *tg.ReplyInlineMarkup {
	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, storage := range stors {
		data := tcbdata.Add{
			Files:            files,
			SelectedStorName: storage.Name(),
			AsBatch:          true,
		}
		dataid := xid.New().String()
		err := cache.Set(dataid, data)
		if err != nil {
			return nil
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: storage.Name(),
			Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dataid),
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

func BuildSetDirKeyboard(dirs []database.Dir, dataid string) (*tg.ReplyInlineMarkup, error) {
	data, ok := cache.Get[tcbdata.Add](dataid)
	if !ok {
		return nil, fmt.Errorf("failed to get data from cache: %s", dataid)
	}
	if data.DirID != 0 || data.SettedDir {
		log.Warnf("Data already has a directory set: %d, %t", data.DirID, data.SettedDir)
		return nil, fmt.Errorf("data already has a directory set")
	}
	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, dir := range dirs {
		dirDataId := xid.New().String()
		dirData := tcbdata.Add{
			Files:            data.Files,
			SelectedStorName: data.SelectedStorName,
			AsBatch:          data.AsBatch,
			DirID:            dir.ID,
			SettedDir:        true,
		}
		err := cache.Set(dirDataId, dirData)
		if err != nil {
			return nil, fmt.Errorf("failed to set directory data in cache: %w", err)
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: dir.Path,
			Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dirDataId),
		})
	}
	dirDefaultDataId := xid.New().String()
	dirDefaultData := tcbdata.Add{
		Files:            data.Files,
		SelectedStorName: data.SelectedStorName,
		AsBatch:          data.AsBatch,
		DirID:            0,
		SettedDir:        true,
	}
	err := cache.Set(dirDefaultDataId, dirDefaultData)
	if err != nil {
		return nil, fmt.Errorf("failed to set default directory data in cache: %w", err)
	}
	buttons = append(buttons, &tg.KeyboardButtonCallback{
		Text: "默认",
		Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dirDefaultDataId),
	})
	markup := &tg.ReplyInlineMarkup{}
	for i := 0; i < len(buttons); i += 3 {
		row := tg.KeyboardButtonRow{}
		row.Buttons = buttons[i:min(i+3, len(buttons))]
		markup.Rows = append(markup.Rows, row)
	}
	return markup, nil
}
