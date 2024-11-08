package bot

import (
	"context"
	"os"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/glebarez/sqlite"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/logger"
)

var Client *gotgproto.Client

func Init() {
	logger.L.Info("Initializing client...")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resultChan := make(chan struct {
		client *gotgproto.Client
		err    error
	})

	go func() {
		client, err := gotgproto.NewClient(int(config.Cfg.Telegram.AppID), config.Cfg.Telegram.AppHash, gotgproto.ClientTypeBot(config.Cfg.Telegram.Token),
			&gotgproto.ClientOpts{
				Session:          sessionMaker.SqlSession(sqlite.Open("data/session.db")),
				DisableCopyright: true,
				Middlewares:      FloodWaitMiddleware(),
			},
		)
		resultChan <- struct {
			client *gotgproto.Client
			err    error
		}{client, err}
	}()

	select {
	case <-ctx.Done():
		logger.L.Fatal("Failed to initialize client")
		os.Exit(1)
	case result := <-resultChan:
		if result.err != nil {
			logger.L.Fatalf("Failed to initialize client: %s", result.err)
			os.Exit(1)
		}
		Client = result.client
		RegisterHandlers(Client.Dispatcher)
		logger.L.Info("Client initialized")
	}
}
