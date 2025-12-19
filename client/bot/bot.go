package bot

import (
	"context"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers"
	"github.com/krau/SaveAny-Bot/client/middleware"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
)

var ectx *ext.Context

func ExtContext() *ext.Context {
	return ectx
}

func Init(ctx context.Context) <-chan struct{} {
	log.FromContext(ctx).Info("Initializing Bot...")
	resultChan := make(chan struct {
		client *gotgproto.Client
		err    error
	})
	shouldRestart := make(chan struct{})

	go func() {
		resolver, err := tgutil.NewConfigProxyResolver()
		if err != nil {
			resultChan <- struct {
				client *gotgproto.Client
				err    error
			}{nil, err}
			return
		}
		client, err := gotgproto.NewClient(
			config.C().Telegram.AppID,
			config.C().Telegram.AppHash,
			gotgproto.ClientTypeBot(config.C().Telegram.Token),
			&gotgproto.ClientOpts{
				Session:          sessionMaker.SqlSession(database.GetDialect(config.C().DB.Session)),
				DisableCopyright: true,
				Middlewares:      middleware.NewDefaultMiddlewares(ctx, 5*time.Minute),
				Resolver:         resolver,
				Context:          ctx,
				MaxRetries:       config.C().Telegram.RpcRetry,
				AutoFetchReply:   true,
				ErrorHandler: func(ctx *ext.Context, u *ext.Update, s string) error {
					if s == "SAVEANTBOT-RESTART" {
						shouldRestart <- struct{}{}
						return dispatcher.EndGroups
					}
					log.FromContext(ctx).Errorf("unhandled error: %s", s)
					return dispatcher.EndGroups
				},
			},
		)
		if err != nil {
			resultChan <- struct {
				client *gotgproto.Client
				err    error
			}{nil, err}
			return
		}
		client.API().BotsSetBotCommands(ctx, &tg.BotsSetBotCommandsRequest{
			Scope: &tg.BotCommandScopeDefault{},
		})
		commands := make([]tg.BotCommand, 0, len(handlers.CommandHandlers))
		for _, info := range handlers.CommandHandlers {
			commands = append(commands, tg.BotCommand{Command: info.Cmd, Description: i18n.T(info.Desc)})
		}
		_, err = client.API().BotsSetBotCommands(ctx, &tg.BotsSetBotCommandsRequest{
			Scope:    &tg.BotCommandScopeDefault{},
			Commands: commands,
		})
		resultChan <- struct {
			client *gotgproto.Client
			err    error
		}{client, err}
	}()

	select {
	case <-ctx.Done():
		log.FromContext(ctx).Errorf("Bot initialization cancelled: %s", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			log.FromContext(ctx).Fatalf("Failed to initialize Bot: %s", result.err)
		}
		handlers.Register(result.client.Dispatcher)
		ectx = result.client.CreateContext()
		log.FromContext(ctx).Info("Bot initialization completed.")
	}
	return shouldRestart
}
