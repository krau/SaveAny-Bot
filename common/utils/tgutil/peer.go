package tgutil

import (
	"github.com/gotd/td/constant"
	"github.com/gotd/td/tg"
	tgstorage "github.com/krau/mygotg/storage"
)

// MarkedChannelID converts a plain channel ID to the Bot API style
// -100<channel_id> form used across the database and HTTP API.
func MarkedChannelID(id int64) int64 {
	var peerID constant.TDLibPeerID
	peerID.Channel(id)
	return int64(peerID)
}

// MarkedChatID converts a plain basic-group ID to its negative Bot API form.
func MarkedChatID(id int64) int64 {
	var peerID constant.TDLibPeerID
	peerID.Chat(id)
	return int64(peerID)
}

// MarkedPeerID converts a plain peer ID to the Bot API style form for its type.
func MarkedPeerID(id int64, typ int) int64 {
	switch tgstorage.EntityType(typ) {
	case tgstorage.TypeChannel:
		return MarkedChannelID(id)
	case tgstorage.TypeChat:
		return MarkedChatID(id)
	default:
		return id
	}
}

// ChatIdFromPeer returns the Bot API style ID of a message or callback peer.
func ChatIdFromPeer(peer tg.PeerClass) int64 {
	switch p := peer.(type) {
	case *tg.PeerChannel:
		return MarkedChannelID(p.ChannelID)
	case *tg.PeerChat:
		return MarkedChatID(p.ChatID)
	case *tg.PeerUser:
		return p.UserID
	default:
		return 0
	}
}
