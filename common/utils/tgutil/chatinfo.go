package tgutil

import (
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
)

// ChatInfoFromExt extracts chat title and username for the given peer.
func ChatInfoFromExt(extCtx *ext.Context, peer tg.PeerClass) (title, username string) {
	if extCtx == nil {
		return
	}
	chatID := ChatIdFromPeer(peer)
	if chatID == 0 {
		return
	}

	if extCtx.Entities != nil {
		if t, u, found := lookupEntities(extCtx.Entities, chatID); found {
			return t, u
		}
	}

	return fetchFromAPI(extCtx, peer, chatID)
}

func formatUserName(firstName, lastName string) string {
	if lastName != "" {
		return firstName + " " + lastName
	}
	return firstName
}

func lookupEntities(entities *tg.Entities, chatID int64) (title, username string, found bool) {
	if ch, ok := entities.Channels[chatID]; ok {
		return ch.Title, ch.Username, true
	}
	if ch, ok := entities.Chats[chatID]; ok {
		return ch.Title, "", true // tg.Chat has no Username field
	}
	if u, ok := entities.Users[chatID]; ok {
		return formatUserName(u.FirstName, u.LastName), u.Username, true
	}
	return "", "", false
}

func fetchFromAPI(extCtx *ext.Context, peer tg.PeerClass, chatID int64) (title, username string) {
	if extCtx.Raw == nil {
		return
	}

	var err error
	switch peer.(type) {
	case *tg.PeerChannel:
		title, username, err = fetchChannel(extCtx, chatID)
	case *tg.PeerChat:
		title, username, err = fetchChat(extCtx, chatID)
	case *tg.PeerUser:
		title, username, err = fetchUser(extCtx, chatID)
	}
	if err != nil {
		log.Debug("Failed to fetch chat info from API", "chatID", chatID, "error", err)
	}
	return
}

func resolveInputPeer(extCtx *ext.Context, chatID int64) (tg.InputPeerClass, error) {
	return extCtx.ResolveInputPeerById(chatID)
}

func fetchChannel(extCtx *ext.Context, chatID int64) (string, string, error) {
	inputPeer, err := resolveInputPeer(extCtx, chatID)
	if err != nil {
		return "", "", err
	}
	ch, ok := inputPeer.(*tg.InputPeerChannel)
	if !ok {
		return "", "", nil
	}

	result, err := extCtx.Raw.ChannelsGetChannels(extCtx, []tg.InputChannelClass{
		&tg.InputChannel{ChannelID: chatID, AccessHash: ch.AccessHash},
	})
	if err != nil {
		return "", "", err
	}
	for _, c := range result.GetChats() {
		if channel, ok := c.(*tg.Channel); ok && channel.ID == chatID {
			return channel.Title, channel.Username, nil
		}
	}
	return "", "", nil
}

func fetchChat(extCtx *ext.Context, chatID int64) (string, string, error) {
	result, err := extCtx.Raw.MessagesGetFullChat(extCtx, chatID)
	if err != nil {
		return "", "", err
	}
	for _, c := range result.GetChats() {
		if chat, ok := c.(*tg.Chat); ok && chat.ID == chatID {
			return chat.Title, "", nil
		}
	}
	return "", "", nil
}

func fetchUser(extCtx *ext.Context, chatID int64) (string, string, error) {
	inputPeer, err := resolveInputPeer(extCtx, chatID)
	if err != nil {
		return "", "", err
	}
	u, ok := inputPeer.(*tg.InputPeerUser)
	if !ok {
		return "", "", nil
	}

	users, err := extCtx.Raw.UsersGetUsers(extCtx, []tg.InputUserClass{
		&tg.InputUser{UserID: chatID, AccessHash: u.AccessHash},
	})
	if err != nil {
		return "", "", err
	}
	for _, user := range users {
		if u, ok := user.(*tg.User); ok && u.ID == chatID {
			return formatUserName(u.FirstName, u.LastName), u.Username, nil
		}
	}
	return "", "", nil
}
