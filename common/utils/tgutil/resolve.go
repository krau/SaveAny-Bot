package tgutil

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/validator"
	"github.com/gotd/td/tg"
)

func ParseChatID(ctx *ext.Context, idOrUsername string) (int64, error) {
	idOrUsername = strings.TrimPrefix(idOrUsername, "@")
	if validator.IsIntStr(idOrUsername) {
		chatID, err := strconv.Atoi(idOrUsername)
		if err != nil {
			return 0, err
		}
		return int64(chatID), nil
	}
	username := idOrUsername
	peer := ctx.PeerStorage.GetPeerByUsername(username)
	if peer != nil && peer.ID != 0 {
		return peer.ID, nil
	}
	chat, err := ctx.ResolveUsername(username)
	if err != nil {
		return 0, err
	}
	if chat == nil {
		return 0, fmt.Errorf("no chat found for username: %s", idOrUsername)
	}
	chatID := chat.GetID()
	if chatID == 0 {
		return 0, fmt.Errorf("chat ID is zero for username: %s", idOrUsername)
	}
	return chatID, nil
}

// return: ChatID, MessageID, error
func ParseMessageLink(ctx *ext.Context, link string) (int64, int, error) {
	u, err := url.Parse(link)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid URL: %w", err)
	}
	paths := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")

	if cmt := u.Query().Get("comment"); cmt != "" {
		// 频道评论的消息链接
		// https://t.me/acherkrau/123?comment=2
		chid, err := ParseChatID(ctx, paths[0])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse chat ID: %w", err)
		}
		chatfull, err := ctx.GetChat(chid)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get chat: %w", err)
		}
		chfull, ok := chatfull.(*tg.ChannelFull)
		if !ok {
			return 0, 0, fmt.Errorf("chat is not a channel: %s", chatfull.TypeName())
		}
		linkChatId, ok := chfull.GetLinkedChatID()
		if !ok {
			return 0, 0, fmt.Errorf("channel has no linked chat")
		}
		msgID, err := strconv.Atoi(cmt)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse comment ID: %w", err)
		}
		return linkChatId, msgID, nil
	}

	switch len(paths) {
	case 2: // https://t.me/acherkrau/123
		chatID, err := ParseChatID(ctx, paths[0])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse chat ID: %w", err)
		}
		msgID, err := strconv.Atoi(paths[1])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse message ID: %w", err)
		}
		return chatID, msgID, nil
	case 3:
		// https://t.me/c/123456789/123
		// https://t.me/acherkrau/123/456 , 123: topic id
		chatPart, msgPart := paths[1], paths[2]
		if paths[0] != "c" {
			chatPart = paths[0]
		}
		chatID, err := ParseChatID(ctx, chatPart)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse chat ID: %w", err)
		}
		msgID, err := strconv.Atoi(msgPart)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse message ID: %w", err)
		}
		return chatID, msgID, nil
	case 4:
		// https://t.me/c/123456789/111/456 111: topic id
		if paths[0] != "c" {
			return 0, 0, fmt.Errorf("invalid message link format: %s", link)
		}
		chatID, err := ParseChatID(ctx, paths[1])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse chat ID: %w", err)
		}
		msgID, err := strconv.Atoi(paths[3])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse message ID: %w", err)
		}
		return chatID, msgID, nil
	}
	return 0, 0, fmt.Errorf("invalid message link format: %s", link)
}
