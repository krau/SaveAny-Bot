package msgelem

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func BuildSelectStorageKeyboard(stors []storage.Storage, file tfile.TGFile) (*tg.ReplyInlineMarkup, error) {
	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, storage := range stors {
		data := tcbdata.Add{
			File:        file,
			StorageName: storage.Name(),
			DirID:       0,
		}
		dataid := xid.New().String()
		err := cache.Set(dataid, data)
		if err != nil {
			return nil, err
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: storage.Name(),
			Data: fmt.Appendf(nil, "add %s", dataid),
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

func BuildSelectStorageMessage(ctx context.Context, stors []storage.Storage, file tfile.TGFile, msgId int) (*tg.MessagesEditMessageRequest, error) {
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
	markup, err := BuildSelectStorageKeyboard(stors, file)
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
			Data: fmt.Appendf(nil, "set_default %s", dataid),
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
