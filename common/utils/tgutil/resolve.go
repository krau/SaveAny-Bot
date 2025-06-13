package tgutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/duke-git/lancet/v2/validator"
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
	chat, err := ctx.ResolveUsername(idOrUsername)
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
