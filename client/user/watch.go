package user

import (
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

type MediaMessageEvent struct {
	Ctx    *ext.Context
	ChatID int64 // from witch the media message was sent
	File   tfile.TGFileMessage
}

var mediaMessageCh = make(chan MediaMessageEvent, 100)

func GetMediaMessageCh() chan MediaMessageEvent {
	return mediaMessageCh
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
	mediaMessageCh <- MediaMessageEvent{
		Ctx:    ctx,
		ChatID: chatId,
		File:   file,
	}
	return dispatcher.EndGroups
}
