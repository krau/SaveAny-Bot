package msgelem

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	msgmarkup "github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func BuildAddSelectStorageKeyboard(stors []storage.Storage, adddata tcbdata.Add) (*tg.ReplyInlineMarkup, error) {
	taskType := adddata.TaskType
	if taskType == "" {
		if len(adddata.Files) > 0 {
			taskType = tasktype.TaskTypeTgfiles
		} else if adddata.TphPageNode != nil {
			taskType = tasktype.TaskTypeTphpics
		} else if adddata.ParsedItem != nil {
			taskType = tasktype.TaskTypeParseditem
		} else {
			return nil, fmt.Errorf("unknown task type: %s", taskType)
		}
	}

	buttons := make([]tg.KeyboardInlineButton, 0)
	for _, st := range stors {
		data := tcbdata.Add{
			TaskType:          taskType,
			SelectedStorageID: storage.ID(st),
			SelectedDirPath:   adddata.SelectedDirPath,
			ConflictStrategy:  adddata.ConflictStrategy,

			Files:   adddata.Files,
			AsBatch: len(adddata.Files) > 1,

			TphPageNode: adddata.TphPageNode,
			TphPics:     adddata.TphPics,
			TphDirPath:  adddata.TphDirPath,

			ParsedItem: adddata.ParsedItem,

			DirectLinks: adddata.DirectLinks,

			Aria2URIs:  adddata.Aria2URIs,
			YtdlpURLs:  adddata.YtdlpURLs,
			YtdlpFlags: adddata.YtdlpFlags,

			TransferSourceStorageID: adddata.TransferSourceStorageID,
			TransferSourcePath:      adddata.TransferSourcePath,
			TransferFiles:           adddata.TransferFiles,
		}
		dataid := xid.New().String()
		err := cache.Set(dataid, data)
		if err != nil {
			return nil, err
		}
		buttons = append(buttons, msgmarkup.Callback(
			st.Name(),
			fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dataid),
		))
	}
	markup := &tg.ReplyInlineMarkup{}
	for i := 0; i < len(buttons); i += 3 {
		row := tg.KeyboardInlineButtonRow{}
		row.Buttons = buttons[i:min(i+3, len(buttons))]
		markup.Rows = append(markup.Rows, row)
	}
	return markup, nil
}

func BuildAddOneSelectStorageMessage(ctx context.Context, stors []storage.Storage, file tfile.TGFileMessage, msgId int) (*tg.MessagesEditMessageRequest, error) {
	eb := entity.Builder{}
	var entities []tg.MessageEntityClass
	text := i18n.T(i18nk.BotMsgTasksInfoAddedToQueueFull, map[string]any{
		"Filename":    file.Name(),
		"QueueLength": 0,
	})
	if err := styling.Perform(&eb,
		styling.Plain(i18n.T(i18nk.BotMsgStorageInfoFilenamePrefix, nil)),
		styling.Code(file.Name()),
		styling.Plain(i18n.T(i18nk.BotMsgStorageInfoPromptSelectStorage, nil)),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entity: %s", err)
	} else {
		text, entities = eb.Complete()
	}
	markup, err := BuildAddSelectStorageKeyboard(stors, tcbdata.Add{
		TaskType: tasktype.TaskTypeTgfiles,
		Files:    []tfile.TGFileMessage{file},
		AsBatch:  false,
	})
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

func BuildConflictStrategyMarkup(adddata tcbdata.Add) (*tg.ReplyInlineMarkup, error) {
	type option struct {
		text     string
		strategy string
	}
	options := []option{
		{text: i18n.T(i18nk.BotMsgCommonButtonConflictRename, nil), strategy: tcbdata.ConflictStrategyRename},
		{text: i18n.T(i18nk.BotMsgCommonButtonConflictOverwrite, nil), strategy: tcbdata.ConflictStrategyOverwrite},
		{text: i18n.T(i18nk.BotMsgCommonButtonConflictSkip, nil), strategy: tcbdata.ConflictStrategySkip},
	}
	buttons := make([]tg.KeyboardInlineButton, 0, len(options))
	for _, opt := range options {
		data := adddata
		data.ConflictStrategy = opt.strategy
		dataid := xid.New().String()
		if err := cache.Set(dataid, data); err != nil {
			return nil, err
		}
		buttons = append(buttons, msgmarkup.Callback(
			opt.text,
			fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dataid),
		))
	}
	rows := make([]tg.KeyboardInlineButtonRow, 0, len(buttons))
	for _, button := range buttons {
		rows = append(rows, tg.KeyboardInlineButtonRow{
			Buttons: []tg.KeyboardInlineButton{button},
		})
	}
	return &tg.ReplyInlineMarkup{Rows: rows}, nil
}

func BuildSetDefaultStorageMarkup(
	ctx context.Context,
	stors []storage.Storage) (*tg.ReplyInlineMarkup, error) {
	buttons := make([]tg.KeyboardInlineButton, 0)
	for _, st := range stors {
		data := tcbdata.SetDefaultStorage{
			StorageID: storage.ID(st),
		}
		dataid := xid.New().String()
		err := cache.Set(dataid, data)
		if err != nil {
			return nil, err
		}
		buttons = append(buttons, msgmarkup.Callback(
			st.Name(),
			fmt.Appendf(nil, "%s %s", tcbdata.TypeSetDefault, dataid),
		))
	}
	markup := &tg.ReplyInlineMarkup{}
	for i := 0; i < len(buttons); i += 3 {
		row := tg.KeyboardInlineButtonRow{}
		row.Buttons = buttons[i:min(i+3, len(buttons))]
		markup.Rows = append(markup.Rows, row)
	}
	return markup, nil
}

func BuildSetDefaultDirMarkup(ctx context.Context,
	storageID uint,
	dirs []database.Dir) (*tg.ReplyInlineMarkup, error) {
	buttons := make([]tg.KeyboardInlineButton, 0)
	for _, dir := range dirs {
		dataid := xid.New().String()
		data := tcbdata.SetDefaultStorage{
			StorageID: storageID,
			DirID:     dir.ID,
		}
		err := cache.Set(dataid, data)
		if err != nil {
			return nil, err
		}
		buttons = append(buttons, msgmarkup.Callback(
			dir.Path,
			fmt.Appendf(nil, "%s %s", tcbdata.TypeSetDefault, dataid),
		))
	}
	markup := &tg.ReplyInlineMarkup{}
	for i := 0; i < len(buttons); i += 3 {
		row := tg.KeyboardInlineButtonRow{}
		row.Buttons = buttons[i:min(i+3, len(buttons))]
		markup.Rows = append(markup.Rows, row)
	}
	return markup, nil
}

func BuildSetDirMarkupForAdd(dirs []database.Dir, dataid string) (*tg.ReplyInlineMarkup, error) {
	data, ok := cache.Get[tcbdata.Add](dataid)
	if !ok {
		return nil, fmt.Errorf("failed to get data from cache: %s", dataid)
	}
	if data.DirID != 0 || data.SettedDir {
		log.Warnf("Data already has a directory set: %d, %t", data.DirID, data.SettedDir)
		return nil, fmt.Errorf("data already has a directory set")
	}
	buttons := make([]tg.KeyboardInlineButton, 0)
	for _, dir := range dirs {
		dirDataId := xid.New().String()
		dirData := data
		dirData.DirID = dir.ID
		dirData.SettedDir = true
		err := cache.Set(dirDataId, dirData)
		if err != nil {
			return nil, fmt.Errorf("failed to set directory data in cache: %w", err)
		}
		buttons = append(buttons, msgmarkup.Callback(
			dir.Path,
			fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dirDataId),
		))
	}
	dirDefaultDataId := xid.New().String()
	dirDefaultData := data
	dirDefaultData.DirID = 0
	dirDefaultData.SettedDir = true
	err := cache.Set(dirDefaultDataId, dirDefaultData)
	if err != nil {
		return nil, fmt.Errorf("failed to set default directory data in cache: %w", err)
	}
	buttons = append(buttons, msgmarkup.Callback(
		i18n.T(i18nk.BotMsgDirButtonDefault, nil),
		fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dirDefaultDataId),
	))
	markup := &tg.ReplyInlineMarkup{}
	for i := 0; i < len(buttons); i += 3 {
		row := tg.KeyboardInlineButtonRow{}
		row.Buttons = buttons[i:min(i+3, len(buttons))]
		markup.Rows = append(markup.Rows, row)
	}
	return markup, nil
}
