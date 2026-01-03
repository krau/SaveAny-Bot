package handlers

import (
	"context"
	"sync"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/storage"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
)

var syncpeerMu sync.Mutex

func handleSyncpeersCmd(ctx *ext.Context, u *ext.Update) error {
	if !config.C().Telegram.Userbot.Enable {
		return dispatcher.EndGroups
	}
	syncpeerMu.Lock()
	defer syncpeerMu.Unlock()
	uctx := user.GetCtx()
	if uctx == nil {
		return dispatcher.EndGroups
	}
	ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgSyncpeersStart)), nil)
	tapi := uctx.Raw
	peerStorage := uctx.PeerStorage
	log.FromContext(ctx).Info("Starting to sync peers...")
	count := 0
	err := dialogs.NewQueryBuilder(tapi).GetDialogs().BatchSize(50).ForEach(ctx, func(ctx context.Context, e dialogs.Elem) error {
		for cid, channel := range e.Entities.Channels() {
			peerStorage.AddPeer(cid, channel.AccessHash, storage.TypeChannel, channel.Username)
			count++
		}
		for uid, user := range e.Entities.Users() {
			peerStorage.AddPeer(uid, user.AccessHash, storage.TypeUser, user.Username)
			count++
		}
		for gid := range e.Entities.Chats() {
			peerStorage.AddPeer(gid, storage.DefaultAccessHash, storage.TypeChat, storage.DefaultUsername)
			count++
		}
		return nil
	})
	if err != nil {
		log.FromContext(ctx).Error("Failed to sync peers", "error", err)
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgSyncpeersFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	log.FromContext(ctx).Info("Finished syncing peers")
	ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgSyncpeersSuccess, map[string]any{
		"Count": count,
	})), nil)
	return dispatcher.EndGroups
}
