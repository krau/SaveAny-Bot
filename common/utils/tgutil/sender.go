package tgutil

import (
	"github.com/gotd/td/tg"
	"github.com/krau/mygotg/ext"
)

// SenderID returns the user responsible for the update. Callback queries carry
// it natively; message updates resolve it from the sender peer.
func SenderID(u *ext.Update) int64 {
	if u == nil {
		return 0
	}
	if u.CallbackQuery != nil {
		return u.CallbackQuery.GetUserID()
	}
	message := u.EffectiveMessage
	// Outgoing updates in private chats belong to the chat peer, not to the
	// bot that sent them.
	if message != nil && message.Out {
		return u.GetUserChat().GetID()
	}
	if user := u.EffectiveUser(); user != nil {
		return user.ID
	}
	if message != nil {
		if from, ok := message.GetFromID(); ok {
			if peer, ok := from.(*tg.PeerUser); ok {
				return peer.UserID
			}
		}
	}
	return u.GetUserChat().GetID()
}

// ChatID returns the chat that replies and edits must target. In private chats
// this equals the sender ID; in groups and supergroups it is the group chat.
// Peer IDs are used directly so callback updates without entity maps still
// resolve their target chat.
func ChatID(u *ext.Update) int64 {
	if u == nil {
		return 0
	}
	if message := u.EffectiveMessage; message != nil {
		if id := ChatIdFromPeer(message.PeerID); id != 0 {
			return id
		}
	}
	if cb := u.CallbackQuery; cb != nil {
		if id := ChatIdFromPeer(cb.Peer); id != 0 {
			return id
		}
		return cb.GetUserID()
	}
	return u.EffectiveChat().GetID()
}
