package bot

import (
	"errors"
	"fmt"
	"net/url"
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
	linkRegexString = `https?://t\.me/(?:c/\d+|[a-zA-Z0-9_]+)/\d+(?:\?[^\s]*)?`
	linkRegex       = regexp.MustCompile(linkRegexString)
)

type parseResult struct {
	ChatID     int64
	MessageID  int
	Files      []*types.File
	UserClient bool
}

func parseLink(ctx *ext.Context, link string) (*parseResult, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("无法解析链接: %s", err)
	}
	strSlice := strings.Split(u.Path, "/")
	if len(strSlice) < 3 {
		return nil, fmt.Errorf("链接格式错误: %s", link)
	}
	messageID, err := strconv.Atoi(strSlice[len(strSlice)-1])
	if err != nil {
		return nil, fmt.Errorf("无法解析消息 ID: %s", err)
	}
	var chatID int64
	if len(strSlice) == 3 {
		chatUsername := strSlice[1]
		peer := ctx.PeerStorage.GetPeerByUsername(chatUsername)
		if peer != nil {
			chatID = peer.ID
		} else {
			linkChat, err := ctx.ResolveUsername(chatUsername)
			if err != nil {
				return nil, fmt.Errorf("解析用户名失败: %s", err)
			}
			if linkChat == nil {
				return nil, fmt.Errorf("找不到该聊天: %s", chatUsername)
			}
			chatID = linkChat.GetID()
		}
	} else if len(strSlice) == 4 {
		chatIDInt, err := strconv.Atoi(strSlice[2])
		if err != nil {
			return nil, fmt.Errorf("无法解析 Chat ID: %s", err)
		}
		chatID = int64(chatIDInt)
	} else {
		return nil, errors.New("链接格式不正确，无法解析 Chat ID")
	}
	if chatID == 0 || messageID == 0 {
		return nil, fmt.Errorf("链接中缺少 Chat ID 或 Message ID: %s", link)
	}
	msg, _, err := tryFetchMessage(ctx, chatID, messageID)
	if err != nil {
		return nil, fmt.Errorf("获取消息失败: %s", err)
	}
	mediaGroup, isGroup := msg.GetGroupedID()
	if u.Query().Has("single") || !isGroup || (mediaGroup == 0) || userclient.UC == nil {
		file, useUserClient, err := tryFetchFileFromMessage(ctx, chatID, messageID, "")
		if err != nil {
			return nil, fmt.Errorf("获取文件失败: %s", err)
		}
		if file.FileName == "" {
			file.FileName = GenFileNameFromMessage(*msg, file)
		}
		return &parseResult{
			ChatID:     chatID,
			MessageID:  messageID,
			Files:      []*types.File{file},
			UserClient: useUserClient,
		}, nil
	}
	groupMessages, isUserClient, err := tryGetMediaGroup(chatID, messageID, mediaGroup)
	if err != nil {
		return nil, fmt.Errorf("获取媒体组消息失败: %s", err)
	}
	var files []*types.File
	for _, groupMsg := range groupMessages {
		file, err := FileFromMedia(groupMsg.Media, "")
		if err != nil {
			return nil, fmt.Errorf("获取媒体文件失败: %s", err)
		}
		if file.FileName == "" {
			file.FileName = GenFileNameFromMessage(*groupMsg, file)
		}
		files = append(files, file)
	}
	return &parseResult{
		ChatID:     chatID,
		MessageID:  messageID,
		Files:      files,
		UserClient: isUserClient,
	}, nil
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

func tryGetMediaGroup(chatID int64, messageID int, mediaGroupID int64) ([]*tg.Message, bool, error) {
	if userclient.UC != nil {
		uctx := userclient.GetCtx()
		messages, err := GetMediaGroup(uctx, chatID, messageID, mediaGroupID)
		if err != nil {
			return nil, true, fmt.Errorf("failed to get media group from userbot: %w", err)
		}
		return messages, true, nil
	}
	return nil, false, errors.New("userclient is not available, cannot fetch media group")
}

func tryFetchMessage(ctx *ext.Context, chatID int64, messageID int) (*tg.Message, bool, error) {
	msg, err := GetTGMessage(ctx, chatID, messageID)
	if err == nil {
		return msg, false, nil
	}
	if  userclient.UC != nil && (strings.Contains(err.Error(), "peer not found") || strings.Contains(err.Error(), "unexpected message type")) {
		common.Log.Warnf("无法获取消息 %d:%d, 尝试使用 userbot: %s", chatID, messageID, err)
		uctx := userclient.GetCtx()
		msg, err := GetTGMessage(uctx, chatID, messageID)
		if err == nil {
			return msg, true, nil
		}
		return nil, true, fmt.Errorf("获取消息失败: %w", err)
	}
	return nil, false, fmt.Errorf("获取消息失败: %s", err)
}

func handleLinkMessage(ctx *ext.Context, update *ext.Update) error {
	common.Log.Trace("Got link message")
	link := linkRegex.FindString(update.EffectiveMessage.Text)
	if link == "" {
		return dispatcher.ContinueGroups
	}
	result, err := parseLink(ctx, link)
	if err != nil {
		common.Log.Errorf("解析链接失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("解析链接失败"), nil)
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

	// TODO: handle group files
	receivedFile := &dao.ReceivedFile{
		Processing:     false,
		FileName:       result.Files[0].FileName,
		ChatID:         result.ChatID,
		MessageID:      result.MessageID,
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
		UseUserClient:  result.UserClient,
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
	file := result.Files[0]
	if !user.Silent || user.DefaultStorage == "" {
		return ProvideSelectMessage(ctx, update, file.FileName, result.ChatID, result.MessageID, replied.ID)
	}
	return HandleSilentAddTask(ctx, update, user, &types.Task{
		Ctx:            ctx,
		Status:         types.Pending,
		FileDBID:       record.ID,
		UseUserClient:  result.UserClient,
		File:           file,
		StorageName:    user.DefaultStorage,
		UserID:         user.ChatID,
		FileChatID:     result.ChatID,
		FileMessageID:  result.MessageID,
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
	})
}
