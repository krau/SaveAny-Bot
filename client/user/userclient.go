package user

import (
	"context"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/dcs"
	"github.com/krau/SaveAny-Bot/client/middleware"
	"github.com/krau/SaveAny-Bot/common/utils/netutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/ncruces/go-sqlite3/gormlite"
	"golang.org/x/net/proxy"
)

var uc *gotgproto.Client
var ectx *ext.Context

func GetCtx() *ext.Context {
	if uc == nil {
		panic("User client is not initialized, please call Login first")
	}
	if ectx != nil {
		return ectx
	}
	ectx = uc.CreateContext()
	return ectx
}

func GetClient() *gotgproto.Client {
	if uc == nil {
		panic("User client is not initialized, please call Login first")
	}
	return uc
}

func Login(ctx context.Context) (*gotgproto.Client, error) {
	log.FromContext(ctx).Debug("Logging in user client")
	if uc != nil {
		return uc, nil
	}
	res := make(chan struct {
		client *gotgproto.Client
		err    error
	})
	go func() {
		var resolver dcs.Resolver
		if config.Cfg.Telegram.Proxy.Enable && config.Cfg.Telegram.Proxy.URL != "" {
			dialer, err := netutil.NewProxyDialer(config.Cfg.Telegram.Proxy.URL)
			if err != nil {
				res <- struct {
					client *gotgproto.Client
					err    error
				}{nil, err}
				return
			}
			resolver = dcs.Plain(dcs.PlainOptions{
				Dial: dialer.(proxy.ContextDialer).DialContext,
			})
		} else {
			resolver = dcs.DefaultResolver()
		}
		tclient, err := gotgproto.NewClient(
			config.Cfg.Telegram.AppID,
			config.Cfg.Telegram.AppHash,
			gotgproto.ClientTypePhone(""),
			&gotgproto.ClientOpts{
				Session:          sessionMaker.SqlSession(gormlite.Open(config.Cfg.Telegram.Userbot.Session)),
				AuthConversator:  &terminalAuthConversator{},
				Context:          ctx,
				DisableCopyright: true,
				Resolver:         resolver,
				MaxRetries:       config.Cfg.Telegram.RpcRetry,
				AutoFetchReply:   true,
				Middlewares:      middleware.NewDefaultMiddlewares(ctx, 5*time.Minute),
				ErrorHandler: func(ctx *ext.Context, u *ext.Update, s string) error {
					log.FromContext(ctx).Errorf("Unhandled error: %s", s)
					return dispatcher.EndGroups
				},
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
		uc = r.client
		log.FromContext(ctx).Infof("User client logged in successfully: %s", uc.Self.FirstName+" "+uc.Self.LastName)
		return uc, nil
	}
}
