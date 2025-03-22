package bot

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/telegraph-go/v2"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/dao"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
)

var (
	TelegraphClient         *telegraph.TelegraphClient
	TelegraphUrlRegexString = `https://telegra.ph/.*`
	TelegraphUrlRegex       = regexp.MustCompile(TelegraphUrlRegexString)
)

func InitTelegraphClient() {
	var httpClient *http.Client
	if config.Cfg.Telegram.Proxy.Enable {
		proxyUrl, err := url.Parse(config.Cfg.Telegram.Proxy.URL)
		if err != nil {
			fmt.Println("Error parsing proxy URL:", err)
			return
		}
		proxy := http.ProxyURL(proxyUrl)
		httpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: proxy,
			},
			Timeout: 30 * time.Second,
		}
	} else {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	TelegraphClient = telegraph.GetTelegraphClient(&telegraph.ClientOpt{HttpClient: httpClient})
}

func handleTelegraph(ctx *ext.Context, update *ext.Update) error {
	common.Log.Trace("Got telegraph link")
	tgphUrl := TelegraphUrlRegex.FindString(update.EffectiveMessage.Text)
	if tgphUrl == "" {
		return dispatcher.ContinueGroups
	}
	replied, err := ctx.Reply(update, ext.ReplyTextString("正在获取文件..."), nil)
	if err != nil {
		common.Log.Errorf("回复失败: %s", err)
		return dispatcher.EndGroups
	}
	user, err := dao.GetUserByChatID(update.GetUserChat().GetID())
	if err != nil {
		common.Log.Errorf("获取用户失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户失败"), nil)
		return dispatcher.EndGroups
	}
	storages := storage.GetUserStorages(user.ChatID)

	if len(storages) == 0 {
		ctx.Reply(update, ext.ReplyTextString("无可用的存储"), nil)
		return dispatcher.EndGroups
	}

	tgphPath := strings.Split(tgphUrl, "/")[len(strings.Split(tgphUrl, "/"))-1]
	fileName, err := url.PathUnescape(tgphPath)
	if err != nil {
		common.Log.Errorf("解析 Telegraph 路径失败: %s", err)
		fileName = tgphPath
	}

	record := &dao.ReceivedFile{
		Processing:     false,
		FileName:       fileName,
		ChatID:         update.EffectiveChat().GetID(),
		MessageID:      update.EffectiveMessage.GetID(),
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.EffectiveChat().GetID(),
		IsTelegraph:    true,
		TelegraphURL:   tgphUrl,
	}
	if err := dao.SaveReceivedFile(record); err != nil {
		common.Log.Errorf("保存接收的文件失败: %s", err)
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			Message: "无法保存文件: " + err.Error(),
			ID:      replied.ID,
		})
		return dispatcher.EndGroups
	}

	if !user.Silent || user.DefaultStorage == "" {
		return ProvideSelectMessage(ctx, update, fileName, update.EffectiveChat().GetID(), update.EffectiveMessage.GetID(), replied.ID)
	}
	return HandleSilentAddTask(ctx, update, user, &types.Task{
		Ctx:            ctx,
		Status:         types.Pending,
		StorageName:    user.DefaultStorage,
		UserID:         user.ChatID,
		ReplyMessageID: replied.ID,
		ReplyChatID:    update.GetUserChat().GetID(),
		IsTelegraph:    true,
		TelegraphURL:   tgphUrl,
	})
}
