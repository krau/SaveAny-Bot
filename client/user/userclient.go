package user

import (
	"context"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/sessionMaker"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/middleware"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
)

var uc *gotgproto.Client
var ectx *ext.Context

func GetCtx() *ext.Context {
	if ectx != nil {
		return ectx
	}
	if uc == nil {
		return nil
	}
	ectx = uc.CreateContext()
	return ectx
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
		resolver, err := tgutil.NewConfigProxyResolver()
		if err != nil {
			res <- struct {
				client *gotgproto.Client
				err    error
			}{nil, err}
			return
		}
		tclient, err := gotgproto.NewClient(
			config.C().Telegram.AppID,
			config.C().Telegram.AppHash,
			gotgproto.ClientTypePhone(""),
			&gotgproto.ClientOpts{
				Session:          sessionMaker.SqlSession(database.GetDialect(config.C().Telegram.Userbot.Session)),
				AuthConversator:  &terminalAuthConversator{},
				Context:          ctx,
				DisableCopyright: true,
				Resolver:         resolver,
				MaxRetries:       config.C().Telegram.RpcRetry,
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
		uc.Dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, func(ctx *ext.Context, u *ext.Update) error {
			switch u.UpdateClass.(type) {
			case *tg.UpdateEditChannelMessage, *tg.UpdateEditMessage, *tg.UpdateDeleteChannelMessages, *tg.UpdateDeleteMessages:
				return dispatcher.EndGroups
			}
			chatId := u.EffectiveChat().GetID()
			watchChats, err := database.GetWatchChatsByChatID(ctx, chatId)
			if err != nil || len(watchChats) == 0 {
				return dispatcher.EndGroups
			}
			return dispatcher.ContinueGroups
		}))
		uc.Dispatcher.AddHandler(handlers.NewMessage(filters.Message.Media, handleMediaMessage))
		log.FromContext(ctx).Infof("User client logged in successfully: %s", uc.Self.FirstName+" "+uc.Self.LastName)
		return uc, nil
	}
}
