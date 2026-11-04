package handlers

import (
	"context"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/mygotg/ext"
	"github.com/krau/mygotg/storage"
)

var syncPeerMu sync.Mutex

// syncPeers imports the userbot's dialogs into the shared peer cache.
func syncPeers(ctx *ext.Context) (int, error) {
	if !config.C().Telegram.Userbot.Enable {
		return 0, nil
	}
	syncPeerMu.Lock()
	defer syncPeerMu.Unlock()
	uctx := user.GetCtx()
	if uctx == nil {
		return 0, nil
	}
	tapi := uctx.Raw
	peerStorage := uctx.PeerStorage
	log.FromContext(ctx).Info("Starting to sync peers...")
	count := 0
	err := dialogs.NewQueryBuilder(tapi).GetDialogs().BatchSize(50).ForEach(ctx, func(ctx context.Context, e dialogs.Elem) error {
		for cid, channel := range e.Entities.Channels() {
			peerStorage.AddPeer(cid, channel.AccessHash, storage.TypeChannel, channel.Username)
			count++
		}
		for uid, usr := range e.Entities.Users() {
			peerStorage.AddPeer(uid, usr.AccessHash, storage.TypeUser, usr.Username)
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
		return count, err
	}
	log.FromContext(ctx).Info("Finished syncing peers")
	return count, nil
}
