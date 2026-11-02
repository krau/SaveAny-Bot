//go:build e2e

package telegrame2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	sabapp "github.com/krau/SaveAny-Bot/app"
	"github.com/krau/SaveAny-Bot/boot"
	"github.com/krau/SaveAny-Bot/client/bot/handlers"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/settings"
	"github.com/krau/mygotg/ext"
	tgstorage "github.com/krau/mygotg/storage"
	"github.com/krau/mygotg/types"
)

const (
	selfID      = int64(999)
	localUserID = int64(10001)
	tgUserID    = int64(10002)
	groupID     = int64(20001)
	channelID   = int64(30001)
)

type Harness struct {
	Server     *Server
	Context    context.Context
	Cancel     context.CancelFunc
	Ext        *ext.Context
	Dispatcher *Dispatcher
	Entities   *tg.Entities
	DataDir    string
	LocalPath  string
	app        *sabapp.App
	oldWD      string
}

func NewHarness() (*Harness, error) {
	dir, err := os.MkdirTemp("", "saveany-telegram-e2e-")
	if err != nil {
		return nil, err
	}
	oldWD, err := os.Getwd()
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	if err := os.Chdir(dir); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	h := &Harness{Context: ctx, Cancel: cancel, DataDir: dir, LocalPath: filepath.Join(dir, "files"), oldWD: oldWD}
	cleanup := true
	defer func() {
		if cleanup {
			cancel()
			_ = os.Chdir(oldWD)
			_ = os.RemoveAll(dir)
		}
	}()

	opts := boot.Options{DataDir: dir, DBPath: filepath.Join(dir, "saveany.db")}
	res, err := boot.Init(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("boot: %w", err)
	}
	cfg := res.Settings.Config()
	cfg.Temp.BasePath = filepath.Join(dir, "cache")
	config.Set(cfg)

	localRecord, err := res.Settings.CreateStorage(ctx, settings.StorageInput{
		Name: "local", Type: "local", Enable: true,
		Config: map[string]any{"base_path": h.LocalPath},
	})
	if err != nil {
		return nil, fmt.Errorf("local storage: %w", err)
	}
	telegramRecord, err := res.Settings.CreateStorage(ctx, settings.StorageInput{
		Name: "tg", Type: "telegram", Enable: true,
		Config: map[string]any{"chat_id": tgUserID},
	})
	if err != nil {
		return nil, fmt.Errorf("telegram storage: %w", err)
	}
	for _, user := range []struct {
		id        int64
		storageID uint
	}{{localUserID, localRecord.ID}, {tgUserID, telegramRecord.ID}} {
		if _, err := database.GetUserByChatID(ctx, user.id); err != nil {
			if !database.IsNotFound(err) {
				return nil, fmt.Errorf("find user %d: %w", user.id, err)
			}
			if err := database.CreateUser(ctx, user.id); err != nil {
				return nil, fmt.Errorf("create user %d: %w", user.id, err)
			}
		}
		record, err := database.GetUserByChatID(ctx, user.id)
		if err != nil {
			return nil, fmt.Errorf("load user %d: %w", user.id, err)
		}
		record.Enabled = true
		record.Silent = true
		record.DefaultStorageID = user.storageID
		if err := database.UpdateUser(ctx, record); err != nil {
			return nil, fmt.Errorf("update user %d: %w", user.id, err)
		}
	}
	if err := res.Settings.ReloadConfig(ctx); err != nil {
		return nil, fmt.Errorf("reload users: %w", err)
	}

	self := &tg.User{ID: selfID, AccessHash: 1, Username: "testbot", FirstName: "Test"}
	self.SetBot(true)
	h.Server = NewServer(self)
	users := map[int64]*tg.User{
		selfID:      self,
		localUserID: {ID: localUserID, AccessHash: 1, FirstName: "Local"},
		tgUserID:    {ID: tgUserID, AccessHash: 1, FirstName: "Telegram"},
	}
	for _, user := range users {
		h.Server.AddUser(user)
	}
	chat := &tg.Chat{ID: groupID, Title: "E2E Group"}
	channel := &tg.Channel{ID: channelID, AccessHash: 1, Title: "E2E Supergroup"}
	channel.SetMegagroup(true)
	h.Server.AddChat(chat)
	h.Server.AddChannel(channel)

	peers := tgstorage.NewPeerStorage(nil, true)
	peers.AddPeer(selfID, 1, tgstorage.TypeUser, "testbot")
	for _, id := range []int64{localUserID, tgUserID} {
		peers.AddPeer(id, 1, tgstorage.TypeUser, "")
	}
	peers.AddPeer(groupID, 0, tgstorage.TypeChat, "")
	peers.AddPeer(channelID, 1, tgstorage.TypeChannel, "")

	h.Entities = &tg.Entities{
		Users:    users,
		Chats:    map[int64]*tg.Chat{groupID: chat},
		Channels: map[int64]*tg.Channel{channelID: channel},
	}
	raw := tg.NewClient(h.Server)
	h.Ext = ext.NewContext(ctx, raw, peers, self, message.NewSender(raw), h.Entities, false)
	h.Dispatcher = NewDispatcher()
	handlers.Register(h.Dispatcher)

	h.app = sabapp.New(ctx, res.Settings, sabapp.Options{
		DataDir: dir,
		DBPath:  filepath.Join(dir, "saveany.db"),
		NoWeb:   true,
	})
	if err := h.app.Start(); err != nil {
		return nil, fmt.Errorf("start app: %w", err)
	}
	cleanup = false
	return h, nil
}

func (h *Harness) Close() {
	if h == nil {
		return
	}
	h.Cancel()
	if h.app != nil {
		h.app.Stop()
	}
	if h.oldWD != "" {
		_ = os.Chdir(h.oldWD)
	}
	if h.DataDir != "" {
		_ = os.RemoveAll(h.DataDir)
	}
}

func (h *Harness) MessageUpdate(message *tg.Message) *ext.Update {
	return &ext.Update{
		UpdateClass:      &tg.UpdateNewMessage{Message: message, Pts: message.ID, PtsCount: 1},
		EffectiveMessage: types.ConstructMessage(message),
		Entities:         h.Entities,
	}
}

func (h *Harness) CommandUpdate(command *tg.Message, replyTo *tg.Message) *ext.Update {
	update := h.MessageUpdate(command)
	if replyTo != nil {
		update.EffectiveMessage.ReplyToMessage = types.ConstructMessage(replyTo)
	}
	return update
}

func (h *Harness) Dispatch(update *ext.Update) error {
	return h.Dispatcher.Dispatch(h.Ext, update)
}

func (h *Harness) UserPeer(id int64) *tg.PeerUser {
	return &tg.PeerUser{UserID: id}
}

func (h *Harness) ClearLocal() error {
	if err := os.RemoveAll(h.LocalPath); err != nil {
		return err
	}
	return os.MkdirAll(h.LocalPath, 0o755)
}

func (h *Harness) WaitLocalFile(timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	for {
		var found []byte
		_ = filepath.WalkDir(h.LocalPath, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || found != nil {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				found = data
			}
			return nil
		})
		if found != nil {
			return found, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("no file found under %s", h.LocalPath)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (h *Harness) WaitFile(relative string, timeout time.Duration) ([]byte, error) {
	path := filepath.Join(h.LocalPath, relative)
	deadline := time.Now().Add(timeout)
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("file %s not found: %w", path, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (h *Harness) WaitUploaded(timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	for {
		if data := h.Server.LastUpload(); len(data) > 0 {
			return data, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("no upload captured")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
