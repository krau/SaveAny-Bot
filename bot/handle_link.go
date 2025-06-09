package bot

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/types"
	"github.com/krau/SaveAny-Bot/userclient"
)

var (
	linkRegexString = `t.me/.*/\d+`
	linkRegex       = regexp.MustCompile(linkRegexString)
)

func parseLink(ctx *ext.Context, link string) (chatID int64, messageID int, err error) {
	strSlice := strings.Split(link, "/")
	if len(strSlice) < 3 {
		return 0, 0, fmt.Errorf("链接格式错误: %s", link)
	}
	messageID, err = strconv.Atoi(strSlice[len(strSlice)-1])
	if err != nil {
		return 0, 0, fmt.Errorf("无法解析消息 ID: %s", err)
	}
	if len(strSlice) == 3 {
		chatUsername := strSlice[1]
		linkChat, err := ctx.ResolveUsername(chatUsername)
		if err != nil {
			return 0, 0, fmt.Errorf("解析用户名失败: %s", err)
		}
		if linkChat == nil {
			return 0, 0, fmt.Errorf("找不到该聊天: %s", chatUsername)
		}
		chatID = linkChat.GetID()
	} else if len(strSlice) == 4 {
		chatIDInt, err := strconv.Atoi(strSlice[2])
		if err != nil {
			return 0, 0, fmt.Errorf("无法解析 Chat ID: %s", err)
		}
		chatID = int64(chatIDInt)
	} else {
		return 0, 0, fmt.Errorf("无效的链接: %s", link)
	}
	return chatID, messageID, nil
}

// use passed ctx client to fetch file from message,
//
// if failed try using userclient
func tryFetchFileFromMessage(ctx *ext.Context, chatID int64, messageID int, fileName string) (*types.File, bool, error) {
	file, err := FileFromMessage(ctx, chatID, messageID, fileName)
	if err == nil {
		return file, false, nil
	}
	if (strings.Contains(err.Error(), "peer not found") || strings.Contains(err.Error(), "unexpected message type")) && userclient.UC != nil {
		common.Log.Warnf("无法获取文件 %d:%d, 尝试使用 userbot: %s", chatID, messageID, err)
		uctx := userclient.GetCtx()
		peer := uctx.PeerStorage.GetInputPeerById(chatID)
		if peer == nil {
			return nil, true, fmt.Errorf("failed to get peer for chat %d: %w", chatID, err)
		}
		msg, err := GetSingleHistoryMessage(uctx, uctx.Raw, peer, messageID)
		if err != nil {
			return nil, true, err
		}
		file, err = FileFromMedia(msg.Media, fileName)
		if err != nil {
			return nil, true, fmt.Errorf("failed to get file from userbot message %d:%d: %w", chatID, messageID, err)
		}
		return file, true, nil
	}
	return nil, false, err
}

func tryFetchMessage(ctx *ext.Context, chatID int64, messageID int) (*tg.Message, error) {
	return GetTGMessage(ctx, chatID, messageID)
}

func handleLinkMessage(ctx *ext.Context, update *ext.Update) error {
	common.Log.Trace("Got link message")
	link := linkRegex.FindString(update.EffectiveMessage.Text)
	if link == "" {
		return dispatcher.ContinueGroups
	}
	linkChatID, messageID, err := parseLink(ctx, link)
	if err != nil {
		common.Log.Errorf("解析链接失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("解析链接失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}

	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		common.Log.Errorf("获取用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}

	replied, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件..."), nil)
	if err != nil {
		common.Log.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}

	file, useUserClient, err := tryFetchFileFromMessage(ctx, linkChatID, messageID, "")
	if err != nil {
		common.Log.Errorf("获取文件失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取文件失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}
	if file.FileName == "" {
		msg, err := tryFetchMessage(ctx, linkChatID, messageID)
		if err != nil {
			file.FileName = fmt.Sprintf("%d_%d", linkChatID, messageID)
		} else {
			file.FileName = GenFileNameFromMessage(*msg, file)
		}
	}

	receivedFile := &dao.ReceivedFile{
		Processing:     false,
		FileName:       file.FileName,
		ChatID:         linkChatID,
		MessageID:      messageID,
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
		UseUserClient:  useUserClient,
	}
	record, err := dao.SaveReceivedFile(receivedFile)
	if err != nil {
		common.Log.Errorf("保存接收的文件失败: %s", err)
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: "无法保存文件: " + err.Error(),
			ID:      replied.ID,
		})
		return dispatcher.EndGroups
	}
	if !user.Silent || user.DefaultStorage == "" {
		return ProvideSelectMessage(ctx, update, file.FileName, linkChatID, messageID, replied.ID)
	}
	return HandleSilentAddTask(ctx, update, user, &types.Task{
		Ctx:            ctx,
		Status:         types.Pending,
		FileDBID:       record.ID,
		UseUserClient:  useUserClient,
		File:           file,
		StorageName:    user.DefaultStorage,
		UserID:         user.ChatID,
		FileChatID:     linkChatID,
		FileMessageID:  messageID,
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
	})
}
