package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot"
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

type MessageContext struct {
	Message *tg.Message
	Client  *ext.Context
}

// 优先使用 Bot，失败后回退到 Userbot
func getClientContext() (*ext.Context, error) {
	if botCtx := bot.ExtContext(); botCtx != nil {
		return botCtx, nil
	}

	if uc := userclient.GetCtx(); uc != nil {
		return uc, nil
	}

	return nil, fmt.Errorf("no client available (bot and userbot are not initialized)")
}

func resolveChatID(_ context.Context, idOrUsername string) (int64, error) {
	if id, err := strconv.ParseInt(idOrUsername, 10, 64); err == nil {
		// 私有频道 ID 需要加上 -100 前缀
		if id > 0 {
			return -1000000000000 - id, nil
		}
		return id, nil
	}

	clientCtx, err := getClientContext()
	if err != nil {
		return 0, err
	}

	return tgutil.ParseChatID(clientCtx, idOrUsername)
}

// 支持的域名: t.me, telegram.me
// 支持格式:
// - https://t.me/username/123
// - https://t.me/c/123456789/123
// - https://t.me/c/123456789/111/456 (topic id)
// - https://t.me/username/123?comment=2 (评论)
func ParseMessageLink(ctx context.Context, link string) (int64, int, error) {
	u, err := url.Parse(link)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid URL: %w", err)
	}
	paths := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")

	if cmt := u.Query().Get("comment"); cmt != "" {
		if len(paths) < 1 {
			return 0, 0, fmt.Errorf("invalid message link format: %s", link)
		}
		return 0, 0, fmt.Errorf("comment links are not supported")
	}

	switch len(paths) {
	case 2: // https://t.me/username/123
		chatID, err := resolveChatID(ctx, paths[0])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to resolve chat ID: %w", err)
		}
		msgID, err := strconv.Atoi(paths[1])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse message ID: %w", err)
		}
		return chatID, msgID, nil
	case 3:
		// https://t.me/c/123456789/123
		// https://t.me/username/123/456 , 123: topic id
		chatPart, msgPart := paths[1], paths[2]
		if paths[0] != "c" {
			chatPart = paths[0]
		}
		chatID, err := resolveChatID(ctx, chatPart)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to resolve chat ID: %w", err)
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
		chatID, err := resolveChatID(ctx, paths[1])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to resolve chat ID: %w", err)
		}
		msgID, err := strconv.Atoi(paths[3])
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse message ID: %w", err)
		}
		return chatID, msgID, nil
	}
	return 0, 0, fmt.Errorf("invalid message link format: %s", link)
}

// 确保消息获取和后续文件创建使用同一个 context
func getMessageWithContext(_ context.Context, chatID int64, msgID int) (*MessageContext, error) {
	if botCtx := bot.ExtContext(); botCtx != nil {
		msg, err := tgutil.GetMessageByID(botCtx, chatID, msgID)
		if err == nil {
			return &MessageContext{Message: msg, Client: botCtx}, nil
		}
	}

	uc := userclient.GetCtx()
	if uc == nil {
		return nil, fmt.Errorf("userbot not initialized and bot cannot access this message")
	}

	msg, err := tgutil.GetMessageByID(uc, chatID, msgID)
	if err != nil {
		return nil, err
	}

	return &MessageContext{Message: msg, Client: uc}, nil
}

// 确保消息获取和后续文件创建使用同一个 context
func getGroupedMessagesWithContext(ctx *MessageContext, chatID int64) ([]*tg.Message, error) {
	msg := ctx.Message
	clientCtx := ctx.Client

	groupID, ok := msg.GetGroupedID()
	if !ok || groupID == 0 {
		return []*tg.Message{msg}, nil
	}

	msgs, err := tgutil.GetGroupedMessages(clientCtx, chatID, msg)
	if err != nil || len(msgs) == 0 {
		// 如果获取失败，至少返回原始消息
		return []*tg.Message{msg}, nil
	}

	return msgs, nil
}

func ExtractFilesFromLinks(ctx context.Context, links []string) ([]tfile.TGFileMessage, error) {
	logger := log.FromContext(ctx)
	var files []tfile.TGFileMessage

	for _, link := range links {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}

		if !isValidMessageLink(link) {
			logger.Errorf("Invalid message link format: %s", link)
			continue
		}

		chatID, msgID, err := ParseMessageLink(ctx, link)
		if err != nil {
			logger.Errorf("Failed to parse message link %s: %v", link, err)
			continue
		}

		u, _ := url.Parse(link)
		single := u != nil && u.Query().Has("single")

		msgCtx, err := getMessageWithContext(ctx, chatID, msgID)
		if err != nil {
			logger.Errorf("Failed to get message %d from chat %d: %v", msgID, chatID, err)
			continue
		}

		msg := msgCtx.Message
		clientCtx := msgCtx.Client

		if msg.Media == nil {
			logger.Warnf("Message %d has no media", msgID)
			continue
		}

		media, ok := msg.GetMedia()
		if !ok {
			logger.Warnf("Failed to get media from message %d", msgID)
			continue
		}

		groupID, isGroup := msg.GetGroupedID()
		if isGroup && groupID != 0 && !single {
			groupMsgs, err := getGroupedMessagesWithContext(msgCtx, chatID)
			if err != nil {
				logger.Errorf("Failed to get grouped messages: %v", err)
			} else {
				for _, gmsg := range groupMsgs {
					if gmsg.Media == nil {
						continue
					}
					gmedia, ok := gmsg.GetMedia()
					if !ok {
						continue
					}
					file, err := tfile.FromMediaMessage(gmedia, clientCtx.Raw, gmsg)
					if err != nil {
						logger.Errorf("Failed to create file from media: %v", err)
						continue
					}
					files = append(files, file)
				}
				continue
			}
		}

		file, err := tfile.FromMediaMessage(media, clientCtx.Raw, msg)
		if err != nil {
			logger.Errorf("Failed to create file from media: %v", err)
			continue
		}
		files = append(files, file)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found in provided links")
	}

	return files, nil
}

func isValidMessageLink(link string) bool {
	for _, prefix := range []string{
		"https://t.me/",
		"http://t.me/",
		"https://telegram.me/",
		"http://telegram.me/",
	} {
		if strings.HasPrefix(link, prefix) {
			return true
		}
	}
	return false
}
