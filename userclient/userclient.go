package userclient

import (
	"context"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/glebarez/sqlite"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/userclient/middlewares"
)

var UC *gotgproto.Client
var ectx *ext.Context

func GetCtx() *ext.Context {
	if ectx != nil {
		return ectx
	}
	ectx = UC.CreateContext()
	return ectx
}

func Login(ctx context.Context) (*gotgproto.Client, error) {
	common.Log.Debug("Logging in as user client")
	if UC != nil {
		return UC, nil
	}
	res := make(chan struct {
		client *gotgproto.Client
		err    error
	})
	go func() {
		tclient, err := gotgproto.NewClient(
			config.Cfg.Telegram.AppID,
			config.Cfg.Telegram.AppHash,
			gotgproto.ClientTypePhone(""),
			&gotgproto.ClientOpts{
				Session:         sessionMaker.SqlSession(sqlite.Open(config.Cfg.Telegram.Userbot.Session)),
				AuthConversator: &termialAuthConversator{},
				// Context:          ctx,
				DisableCopyright: true,
				Middlewares:      middlewares.NewDefaultMiddlewares(ctx, 5*time.Minute),
			},
		)
		if err != nil {
			res <- struct {
				client *gotgproto.Client
				err    error
			}{nil, err}
		}
		res <- struct {
			client *gotgproto.Client
			err    error
		}(struct {
			client *gotgproto.Client
			err    error
		}{tclient, nil})
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-res:
		if r.err != nil {
			return nil, r.err
		}
		UC = r.client
		return UC, nil
	}
}
