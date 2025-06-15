package bot

import (
	"context"
	"net/url"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers"
	"github.com/krau/SaveAny-Bot/client/middleware"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/ncruces/go-sqlite3/gormlite"
	"golang.org/x/net/proxy"
)

var Client *gotgproto.Client

func newProxyDialer(proxyUrl string) (proxy.Dialer, error) {
	url, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}
	return proxy.FromURL(url, proxy.Direct)
}

func Init(ctx context.Context) {
	log.FromContext(ctx).Info("初始化 Bot...")
	resultChan := make(chan struct {
		client *gotgproto.Client
		err    error
	})
	go func() {
		var resolver dcs.Resolver
		if config.Cfg.Telegram.Proxy.Enable && config.Cfg.Telegram.Proxy.URL != "" {
			dialer, err := newProxyDialer(config.Cfg.Telegram.Proxy.URL)
			if err != nil {
				resultChan <- struct {
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
		client, err := gotgproto.NewClient(config.Cfg.Telegram.AppID,
			config.Cfg.Telegram.AppHash,
			gotgproto.ClientTypeBot(config.Cfg.Telegram.Token),
			&gotgproto.ClientOpts{
				Session:          sessionMaker.SqlSession(gormlite.Open(config.Cfg.DB.Session)),
				DisableCopyright: true,
				Middlewares:      middleware.NewDefaultMiddlewares(ctx, 5*time.Minute),
				Resolver:         resolver,
				Context:          ctx,
				MaxRetries:       config.Cfg.Telegram.RpcRetry,
				AutoFetchReply:   true,
				ErrorHandler: func(ctx *ext.Context, u *ext.Update, s string) error {
					log.FromContext(ctx).Errorf("Unhandled error: %s", s)
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
		_, err = client.API().BotsSetBotCommands(ctx, &tg.BotsSetBotCommandsRequest{
			Scope: &tg.BotCommandScopeDefault{},
			Commands: []tg.BotCommand{
				{Command: "start", Description: "开始使用"},
				{Command: "help", Description: "显示帮助"},
				{Command: "silent", Description: "开启/关闭静默模式"},
				{Command: "storage", Description: "设置默认存储端"},
				{Command: "save", Description: "保存所回复的文件"},
				{Command: "dir", Description: "管理存储文件夹"},
				{Command: "rule", Description: "管理规则"},
			},
		})
		resultChan <- struct {
			client *gotgproto.Client
			err    error
		}{client, err}
	}()

	select {
	case <-ctx.Done():
		log.FromContext(ctx).Errorf("已取消 Bot 初始化: %s", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			log.FromContext(ctx).Fatalf("初始化 Bot 失败: %s", result.err)
		}
		Client = result.client
		handlers.Register(Client.Dispatcher)
		log.FromContext(ctx).Info("Bot 初始化完成")
	}
}
