package bot

import (
	"context"
	"net/url"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
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
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Cfg.Telegram.Timeout)*time.Second)
	defer cancel()
	go InitTelegraphClient()
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
				Middlewares:      FloodWaitMiddleware(),
				Resolver:         resolver,
				Context:          ctx,
				MaxRetries:       config.Cfg.Telegram.RpcRetry,
			},
		)
		if err != nil {
			resultChan <- struct {
				client *gotgproto.Client
				err    error
			}{nil, err}
			return
		}
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
	case <-timeoutCtx.Done():
		log.FromContext(ctx).Errorf("初始化 Bot 超时")
	case result := <-resultChan:
		if result.err != nil {
			log.FromContext(ctx).Fatalf("初始化 Bot 失败: %s", result.err)
		}
		Client = result.client
		RegisterHandlers(Client.Dispatcher)
		log.FromContext(ctx).Info("Bot 初始化完成")
	}
}
