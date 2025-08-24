package handlers

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

func handleUpdateCmd(ctx *ext.Context, u *ext.Update) error {
	currentV, err := semver.Parse(config.Version)
	if err != nil {
		ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("You are in dev or the version var failed to inject: %v", err)), nil)
		return dispatcher.EndGroups
	}
	latest, ok, err := selfupdate.DetectLatest(config.GitRepo)
	if err != nil {
		ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("检测最新版本失败: %v", err)), nil)
		return dispatcher.EndGroups
	}
	if !ok {
		ctx.Reply(u, ext.ReplyTextString("没有找到版本信息"), nil)
		return dispatcher.EndGroups
	}
	if latest.Version.LT(currentV) || latest.Version.Equals(currentV) {
		ctx.Reply(u, ext.ReplyTextString(fmt.Sprintf("当前已经是最新版本: %s", config.Version)), nil)
		return dispatcher.EndGroups
	}
	ctx.Sender.To(u.GetUserChat().AsInputPeer()).StyledText(ctx, html.String(nil, func() string {
		md := latest.ReleaseNotes
		md = regexp.MustCompile(`(?m)^###\s+&nbsp;&nbsp;&nbsp;(.+)$`).ReplaceAllString(md, "<b>$1</b>")
		md = regexp.MustCompile(`(?m)^#####\s+&nbsp;&nbsp;&nbsp;&nbsp;(.+)$`).ReplaceAllString(md, "<i>$1</i>")

		md = regexp.MustCompile(`(?m)^- `).ReplaceAllString(md, "• ")

		md = regexp.MustCompile(`\[\((\w{6,})\)\]\((https?://[^\s)]+)\)`).ReplaceAllString(md, `(<a href="$2">$1</a>)`)

		md = regexp.MustCompile(`\[(.+?)\]\((https?://[^\s)]+)\)`).ReplaceAllString(md, `<a href="$2">$1</a>`)

		md = strings.ReplaceAll(md, "&nbsp;", " ")

		return `<blockquote expandable>` + md + `</blockquote>`
	}()))
	text := fmt.Sprintf(`发现新版本: %s
当前版本: %s

文件大小: %.2f MB
下载链接: %s
发布时间: %s

升级将重启 Bot , 是否升级?`, latest.Version, config.Version,
		float64(latest.AssetByteSize)/(1024*1024), latest.AssetURL,
		latest.PublishedAt.Format("2006-01-02 15:04:05"),
	)
	ctx.Reply(u, ext.ReplyTextString(text), &ext.ReplyOpts{
		Markup: &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "升级",
							Data: []byte("update"),
						},
					},
				},
			},
		},
	})
	return dispatcher.EndGroups
}

func handleUpdateCallback(ctx *ext.Context, u *ext.Update) error {
	currentV, err := semver.Parse(config.Version)
	if err != nil {
		return err
	}
	ctx.EditMessage(u.GetUserChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:      u.CallbackQuery.GetMsgID(),
		Message: fmt.Sprintf("正在升级中, 当前版本: %s", config.Version),
	})
	latest, err := selfupdate.UpdateSelf(currentV, config.GitRepo)
	if err != nil {
		ctx.EditMessage(u.GetUserChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:      u.CallbackQuery.GetMsgID(),
			Message: fmt.Sprintf("升级失败: %v", err),
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(u.GetUserChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:      u.CallbackQuery.GetMsgID(),
		Message: fmt.Sprintf("已升级至版本 %s\n若 Bot 未自动重启请手动启动", latest.Version),
	})
	return errors.New("SAVEANTBOT-RESTART")
}
