package handlers

import (
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/mygotg/ext"
)

func syncPeers(ctx *ext.Context) (int, error) {
	return userclient.SyncPeers(ctx)
}
