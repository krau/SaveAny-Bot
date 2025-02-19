package bot

import (
"context"
"net/url"
"os"
"time"

"github.com/celestix/gotgproto"
"github.com/celestix/gotgproto/sessionMaker"
"github.com/glebarez/sqlite"
"github.com/gotd/td/telegram/dcs"
"github.com/gotd/td/tg"
"github.com/krau/SaveAny-Bot/config"
"github.com/krau/SaveAny-Bot/logger"
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

func Init() {
logger.L.Info("Initializing Telegram client...")
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()
resultChan := make(chan struct {
client *gotgproto.Client
err error
})
go func() {
var resolver dcs.Resolver
if config.Cfg.Telegram.Proxy.Enable && config.Cfg.Telegram.Proxy.URL != "" {
dialer, err := newProxyDialer(config.Cfg.Telegram.Proxy.URL)
if err != nil {
resultChan <- struct {
client *gotgproto.Client
err error
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
Session: sessionMaker.SqlSession(sqlite.Open("data/session.db")),
DisableCopyright: true,
Middlewares: FloodWaitMiddleware(),
Resolver: resolver,
},
)
if err != nil {
resultChan <- struct {
client *gotgproto.Client
err error
}{nil, err}
return
}
_, err = client.API().BotsSetBotCommands(ctx, &tg.BotsSetBotCommandsRequest{
Scope: &tg.BotCommandScopeDefault{},
Commands: []tg.BotCommand{
{Command: "start", Description: "Start using"},
{Command: "help", Description: "Show help"},
{Command: "silent", Description: "Turn on/off silent mode"},
{Command: "storage", Description: "Set the default storage"},
{Command: "save", Description: "Save the replied file"},
},
})
resultChan <- struct {
client *gotgproto.Client
err error
}{client, err}
}()

select {
case <-ctx.Done():
logger.L.Fatal("Failed to initialize client: timeout")
os.Exit(1)
case result := <-resultChan:
if result.err != nil {
logger.L.Fatalf("Failed to initialize client: %s", result.err)
os.Exit(1)
}
Client = result.client
RegisterHandlers(Client.Dispatcher)
logger.L.Info("Client initialization completed")
}
}
