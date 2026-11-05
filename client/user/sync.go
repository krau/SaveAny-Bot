package user

import (
	"context"
	"errors"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/mygotg/storage"
)

var syncPeersMu sync.Mutex

// SyncPeers imports the userbot's dialogs into the shared peer cache.
func SyncPeers(ctx context.Context) (int, error) {
	if !config.C().Telegram.Userbot.Enable {
		return 0, errors.New("userbot is not enabled")
	}
	syncPeersMu.Lock()
	defer syncPeersMu.Unlock()
	uctx := GetCtx()
	if uctx == nil {
		return 0, errors.New("userbot is not running")
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
		return count, err
	}
	log.FromContext(ctx).Info("Finished syncing peers")
	return count, nil
}
