package user

import (
	"sync"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

type MediaMessageEvent struct {
	Ctx       *ext.Context
	ChatID    int64 // from witch the media message was sent
	MessageID int
	File      tfile.TGFileMessage
}

type messageKey struct {
	ChatID    int64
	MessageID int
}

type MediaMessageHandler struct {
	events   map[messageKey]MediaMessageEvent
	timers   map[messageKey]*time.Timer
	mu       sync.Mutex
	debounce time.Duration
}

var (
	mediaMessageCh      = make(chan MediaMessageEvent, 100)
	mediaMessageHandler = &MediaMessageHandler{
		events:   make(map[messageKey]MediaMessageEvent),
		timers:   make(map[messageKey]*time.Timer),
		debounce: 5 * time.Second,
	}
)

func GetMediaMessageCh() chan MediaMessageEvent {
	return mediaMessageCh
}

func sendMediaMessageEvent(event MediaMessageEvent) {
	key := messageKey{ChatID: event.ChatID, MessageID: event.MessageID}

	mediaMessageHandler.mu.Lock()
	defer mediaMessageHandler.mu.Unlock()

	if timer, exists := mediaMessageHandler.timers[key]; exists {
		timer.Stop()
	} else {
		mediaMessageHandler.events[key] = event
	}

	mediaMessageHandler.timers[key] = time.AfterFunc(mediaMessageHandler.debounce, func() {
		mediaMessageHandler.mu.Lock()
		event := mediaMessageHandler.events[key]
		delete(mediaMessageHandler.events, key)
		delete(mediaMessageHandler.timers, key)
		mediaMessageHandler.mu.Unlock()

		mediaMessageCh <- event
	})
}

func handleMediaMessage(ctx *ext.Context, update *ext.Update) error {
	message := update.EffectiveMessage
	media, ok := message.GetMedia()
	if !ok || media == nil {
		return dispatcher.EndGroups
	}
	support := func() bool {
		switch media.(type) {
		case *tg.MessageMediaDocument, *tg.MessageMediaPhoto:
			return true
		default:
			return false
		}
	}()
	if !support {
		return dispatcher.EndGroups
	}
	file, err := tfile.FromMediaMessage(media, ctx.Raw, message.Message, tfile.WithNameIfEmpty(
		tgutil.GenFileNameFromMessage(*message.Message),
	))
	if err != nil {
		return err
	}
	chatId := update.EffectiveChat().GetID()
	sendMediaMessageEvent(MediaMessageEvent{
		Ctx:       ctx,
		ChatID:    chatId,
		MessageID: message.ID,
		File:      file,
	})
	return dispatcher.EndGroups
}
