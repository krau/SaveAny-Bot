package tgutil

import (
	"sync"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/constant"
	"github.com/gotd/td/tg"
	"github.com/krau/mygotg/ext"
)

// ChatInfo is the normalized display identity of a Telegram peer.
type ChatInfo struct {
	ID       int64  `json:"id"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

var chatInfoCache sync.Map // map[int64]ChatInfo

// ChatInfoFromExt resolves a peer's title and username from update entities,
// then from the peer cache/API. Empty fields are left empty.
func ChatInfoFromExt(ctx *ext.Context, peer tg.PeerClass) ChatInfo {
	id := ChatIdFromPeer(peer)
	if id == 0 {
		return ChatInfo{}
	}
	if cached, ok := chatInfoCache.Load(id); ok {
		return cached.(ChatInfo)
	}
	info := ChatInfo{ID: id}
	if ctx != nil {
		plain := constant.TDLibPeerID(id).ToPlain()
		if ctx.Entities != nil {
			switch {
			case ctx.Entities.Channels[plain] != nil:
				channel := ctx.Entities.Channels[plain]
				info.Title, info.Username = channel.Title, channel.Username
			case ctx.Entities.Chats[plain] != nil:
				info.Title = ctx.Entities.Chats[plain].Title
			case ctx.Entities.Users[plain] != nil:
				user := ctx.Entities.Users[plain]
				info.Title, info.Username = displayName(user), user.Username
			}
		}
		if info.Title == "" && ctx.Raw != nil {
			info.Title, info.Username = fetchChatInfo(ctx, peer, plain)
		}
	}
	if info.Title != "" || info.Username != "" {
		chatInfoCache.Store(id, info)
	}
	return info
}

func displayName(user *tg.User) string {
	if user == nil {
		return ""
	}
	if user.LastName != "" {
		return user.FirstName + " " + user.LastName
	}
	return user.FirstName
}

func fetchChatInfo(ctx *ext.Context, peer tg.PeerClass, plainID int64) (string, string) {
	switch peer.(type) {
	case *tg.PeerChannel:
		return fetchChannelInfo(ctx, plainID)
	case *tg.PeerChat:
		return fetchGroupInfo(ctx, plainID)
	case *tg.PeerUser:
		return fetchUserInfo(ctx, plainID)
	}
	return "", ""
}

func fetchChannelInfo(ctx *ext.Context, channelID int64) (string, string) {
	peer, err := ctx.ResolveInputPeerById(MarkedChannelID(channelID))
	if err != nil {
		log.Debugf("Failed to resolve channel %d: %v", channelID, err)
		return "", ""
	}
	channelPeer, ok := peer.(*tg.InputPeerChannel)
	if !ok {
		return "", ""
	}
	result, err := ctx.Raw.ChannelsGetChannels(ctx, []tg.InputChannelClass{
		&tg.InputChannel{ChannelID: channelID, AccessHash: channelPeer.AccessHash},
	})
	if err != nil {
		log.Debugf("Failed to fetch channel %d: %v", channelID, err)
		return "", ""
	}
	for _, item := range result.GetChats() {
		if channel, ok := item.(*tg.Channel); ok && channel.ID == channelID {
			return channel.Title, channel.Username
		}
	}
	return "", ""
}

func fetchGroupInfo(ctx *ext.Context, chatID int64) (string, string) {
	result, err := ctx.Raw.MessagesGetFullChat(ctx, chatID)
	if err != nil {
		log.Debugf("Failed to fetch chat %d: %v", chatID, err)
		return "", ""
	}
	for _, item := range result.GetChats() {
		if chat, ok := item.(*tg.Chat); ok && chat.ID == chatID {
			return chat.Title, ""
		}
	}
	return "", ""
}

func fetchUserInfo(ctx *ext.Context, userID int64) (string, string) {
	peer, err := ctx.ResolveInputPeerById(userID)
	if err != nil {
		log.Debugf("Failed to resolve user %d: %v", userID, err)
		return "", ""
	}
	userPeer, ok := peer.(*tg.InputPeerUser)
	if !ok {
		return "", ""
	}
	users, err := ctx.Raw.UsersGetUsers(ctx, []tg.InputUserClass{
		&tg.InputUser{UserID: userID, AccessHash: userPeer.AccessHash},
	})
	if err != nil {
		log.Debugf("Failed to fetch user %d: %v", userID, err)
		return "", ""
	}
	for _, item := range users {
		if user, ok := item.(*tg.User); ok && user.ID == userID {
			return displayName(user), user.Username
		}
	}
	return "", ""
}
