package bot

import (
	"encoding/json"
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/logger"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

func manageStorageEntry(ctx *ext.Context, update *ext.Update) error {
	user, err := dao.GetUserByChatID(update.EffectiveUser().GetID())
	if err != nil {
		logger.L.Errorf("Failed to get user active storages: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户存储失败"), nil)
		return dispatcher.EndGroups
	}

	state, ok := userConversationState[user.ChatID]
	if !ok {
		state = &ConversationState{}
		userConversationState[user.ChatID] = state
	}
	state.Reset()
	state.InConversation = true
	state.SetConversationType(ConversationTypeManageStorage)
	state.SetData("status", "entry")

	storagesMsg := "已添加的存储:"
	if len(user.Storages) == 0 {
		storagesMsg += " 无"
	} else {
		for i, storage := range user.Storages {
			storagesMsg += fmt.Sprintf("\n%d. %s", i+1, storage.Name)
		}
	}
	storagesMsg += "\n\n请选择操作:"
	_, err = ctx.Reply(update, ext.ReplyTextString(storagesMsg), &ext.ReplyOpts{
		Markup: &manageStorageKeyboardMarkup,
	})
	if err != nil {
		logger.L.Errorf("Failed to send manage storage message: %s", err)
		return dispatcher.EndGroups
	}

	return dispatcher.EndGroups
}

func handleManageStorageConversation(ctx *ext.Context, update *ext.Update, state *ConversationState) error {
	status := state.GetData("status").(string)
	switch status {
	case "entry":
		return manageStorageMenu(ctx, update, state)
	case "add_select_type":
		return manageStorageAddSelectType(ctx, update, state)
	case "selected_add_type":
		return manageStorageAddSelectedType(ctx, update, state)
	default:
		logger.L.Errorf("Unknown manage storage status: %s", status)
	}
	return dispatcher.EndGroups
}

func manageStorageMenu(ctx *ext.Context, update *ext.Update, state *ConversationState) error {
	text := update.EffectiveMessage.Text
	switch text {
	case manageStorageButtonAdd:
		return manageStorageAdd(ctx, update, state)
	case manageStorageButtonDelete:
		return manageStorageDelete(ctx, update)
	case manageStorageButtonEdit:
		return manageStorageEdit(ctx, update)
	case manageStorageButtonSetDefault:
		return manageStorageSetDefault(ctx, update)
	default:
		logger.L.Errorf("Unknown manage storage button: %s", text)
		ctx.Reply(update, ext.ReplyTextString("未知操作"), nil)
		return dispatcher.EndGroups
	}
}

func manageStorageAdd(ctx *ext.Context, update *ext.Update, state *ConversationState) error {
	rows := make([]tg.KeyboardButtonRow, 0)
	buttons := make([]tg.KeyboardButtonClass, 0)
	for i, storageType := range types.StorageTypes {
		buttons = append(buttons, &tg.KeyboardButton{
			Text: types.StorageTypeDisplay[storageType],
		})
		if (i+1)%3 == 0 || i == len(types.StorageTypes)-1 {
			rows = append(rows, tg.KeyboardButtonRow{
				Buttons: buttons,
			})
			buttons = make([]tg.KeyboardButtonClass, 0)
		}
	}
	manageStorageAddKeyboardMarkup := tg.ReplyKeyboardMarkup{
		Selective: true,
		Resize:    true,
		Rows:      rows,
	}

	state.SetData("status", "add_select_type")

	ctx.Reply(update, ext.ReplyTextString("请选择要添加的存储类型"), &ext.ReplyOpts{
		Markup: &manageStorageAddKeyboardMarkup,
	})
	return dispatcher.ContinueGroups
}

func manageStorageAddSelectType(ctx *ext.Context, update *ext.Update, state *ConversationState) error {
	text := update.EffectiveMessage.Text
	var storageType types.StorageType
	for t, display := range types.StorageTypeDisplay {
		if display == text {
			storageType = t
			break
		}
	}
	if storageType == "" {
		ctx.Reply(update, ext.ReplyTextString("未知的存储类型"), nil)
		return dispatcher.EndGroups
	}
	state.SetData("status", "selected_add_type")
	state.SetData("storage_type", storageType)
	return manageStorageAddSelectedType(ctx, update, state)
}

func manageStorageAddSelectedType(ctx *ext.Context, update *ext.Update, state *ConversationState) error {
	selectedType := state.GetData("storage_type").(types.StorageType)
	configItems := storage.GetStorageConfigurableItems(selectedType)
	configIndexData := state.GetData("configindex")
	configIndex := 0
	if configIndexData == nil {
		state.SetData("configindex", configIndex)
	} else {
		configIndex = configIndexData.(int)
		if update.EffectiveMessage.Text != "" {
			logger.L.Debugf("config %s: %s", configItems[configIndex-1], update.EffectiveMessage.Text)
			state.SetData(configItems[configIndex-1], update.EffectiveMessage.Text)
		}
	}
	if configIndex >= len(configItems) {
		// TODO: save storage
		state.SetData("status", "add_complete")
		logger.L.Infof("Save storage")
		return manageStorageSave(ctx, update, state)
	}

	ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("正在配置 %s 存储...\n请提供 %s", types.StorageTypeDisplay[selectedType], configItems[configIndex])), &ext.ReplyOpts{
		Markup: &tg.ReplyKeyboardForceReply{
			Selective: true,
			SingleUse: true,
		},
	})
	state.SetData("configindex", configIndex+1)
	return dispatcher.EndGroups
}

func manageStorageSave(ctx *ext.Context, update *ext.Update, state *ConversationState) error {
	defer func() {
		if r := recover(); r != nil {
			logger.L.Errorf("Failed to save storage: %s", r)
			ctx.Reply(update, ext.ReplyTextString("存储配置失败"), nil)
		}
		state.Reset()
	}()
	storageType := state.GetData("storage_type").(types.StorageType)
	config := make(map[string]string)
	configItems := storage.GetStorageConfigurableItems(storageType)
	for _, item := range configItems {
		config[item] = state.GetData(item).(string)
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		logger.L.Errorf("Failed to marshal storage config: %s", err)
		ctx.Reply(update, ext.ReplyTextString("存储配置失败"), nil)
		return dispatcher.EndGroups
	}
	user, err := dao.GetUserByChatID(update.EffectiveUser().GetID())
	if err != nil {
		logger.L.Errorf("Failed to get user: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}
	storageModel := types.StorageModel{
		Type:   string(storageType),
		Active: true,
		Config: configJSON,
	}
	hash := storageModel.GenHash()
	storageModel.Hash = hash
	if storagedb, err := dao.GetStorageByHash(hash); err == nil {
		logger.L.Debugf("Storage already exists")
		user.Storages = append(user.Storages, storagedb)
	} else {
		if id, err := dao.CreateStorage(&storageModel); err != nil {
			logger.L.Errorf("Failed to create storage: %s", err)
			ctx.Reply(update, ext.ReplyTextString("存储创建失败"), nil)
			return dispatcher.EndGroups
		} else {
			storagedb := &types.StorageModel{}
			storagedb.ID = id
			user.Storages = append(user.Storages, storagedb)
		}
	}
	if err := dao.UpdateUser(user); err != nil {
		logger.L.Errorf("Failed to update user with storages: %s", err)
		ctx.Reply(update, ext.ReplyTextString("用户更新失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString("存储已添加"), &ext.ReplyOpts{
		Markup: &tg.ReplyKeyboardHide{},
	})
	return dispatcher.EndGroups
}

func manageStorageDelete(ctx *ext.Context, update *ext.Update) error {
	return dispatcher.ContinueGroups
}

func manageStorageEdit(ctx *ext.Context, update *ext.Update) error {
	return dispatcher.ContinueGroups
}

func manageStorageSetDefault(ctx *ext.Context, update *ext.Update) error {
	return dispatcher.ContinueGroups
}
