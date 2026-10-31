package database

import (
	"context"

	"github.com/gotd/td/constant"
)

func (user *User) WatchChat(ctx context.Context, chat WatchChat) error {
	if len(user.WatchChats) == 0 {
		user.WatchChats = make([]WatchChat, 0)
	}

	user.WatchChats = append(user.WatchChats, chat)
	return db.WithContext(ctx).Save(user.WatchChats).Error
}

func (user *User) UnwatchChat(ctx context.Context, chatID int64) error {
	var watchChat WatchChat
	err := db.WithContext(ctx).Where("chat_id = ? AND user_id = ?", chatID, user.ID).First(&watchChat).Error
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Unscoped().Delete(&watchChat).Error
}

func (user *User) WatchingChat(ctx context.Context, chatID int64) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&WatchChat{}).Where("chat_id = ? AND user_id = ?", chatID, user.ID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetWatchChatsByChatID(ctx context.Context, chatID int64) ([]*WatchChat, error) {
	var watchChats []*WatchChat
	err := db.WithContext(ctx).Where("chat_id = ?", chatID).Find(&watchChats).Error
	if err != nil {
		return nil, err
	}
	return watchChats, nil
}

// MigrateLegacyPeerIDs converts TDLib-encoded watch chat IDs written by
// gotgproto to the plain IDs used by mygotg.
func MigrateLegacyPeerIDs(ctx context.Context) error {
	var chats []WatchChat
	if err := db.WithContext(ctx).Where("chat_id < 0").Find(&chats).Error; err != nil {
		return err
	}
	for _, chat := range chats {
		plain := constant.TDLibPeerID(chat.ChatID).ToPlain()
		if plain == 0 || plain == chat.ChatID {
			continue
		}
		if err := db.WithContext(ctx).Model(&WatchChat{}).
			Where("id = ?", chat.ID).
			Update("chat_id", plain).Error; err != nil {
			return err
		}
	}
	return nil
}
