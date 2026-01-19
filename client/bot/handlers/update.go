package handlers

import (
	"errors"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/i18n"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/unvgo/ghselfupdate"
)

func handleUpdateCmd(ctx *ext.Context, u *ext.Update) error {
	currentV, err := semver.Parse(config.Version)
	if err != nil {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgUpdateErrorVersionVarInvalid, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	latest, ok, err := ghselfupdate.DetectLatest(config.GitRepo)
	if err != nil {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgUpdateErrorCheckLatestFailed, map[string]any{
			"Error": err.Error(),
		})), nil)
		return dispatcher.EndGroups
	}
	if !ok {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgUpdateErrorNoReleaseFound, nil)), nil)
		return dispatcher.EndGroups
	}
	if latest.Version.Major != currentV.Major {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgUpdateInfoMajorUpgradeRequired, map[string]any{
			"Current": currentV.String(),
			"Latest":  latest.Version.String(),
		})), nil)
		return dispatcher.EndGroups
	}
	if latest.Version.LT(currentV) || latest.Version.Equals(currentV) {
		ctx.Reply(u, ext.ReplyTextString(i18n.T(i18nk.BotMsgUpdateInfoAlreadyLatest, map[string]any{
			"Version": config.Version,
		})), nil)
		return dispatcher.EndGroups
	}
	indocker := config.Docker == "true"
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
	if indocker {
		text := i18n.T(i18nk.BotMsgUpdateInfoNewVersionInDocker, map[string]any{
			"Latest":      latest.Version.String(),
			"Current":     config.Version,
			"PublishedAt": latest.PublishedAt.Format("2006-01-02 15:04:05"),
		})
		ctx.Reply(u, ext.ReplyTextString(text), nil)
		return dispatcher.EndGroups
	}
	text := i18n.T(i18nk.BotMsgUpdateInfoNewVersionPromptUpgrade, map[string]any{
		"Latest":      latest.Version.String(),
		"Current":     config.Version,
		"SizeMB":      float64(latest.AssetByteSize) / (1024 * 1024),
		"URL":         latest.AssetURL,
		"PublishedAt": latest.PublishedAt.Format("2006-01-02 15:04:05"),
	})
	ctx.Reply(u, ext.ReplyTextString(text), &ext.ReplyOpts{
		Markup: &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: i18n.T(i18nk.BotMsgUpdateButtonUpgrade, nil),
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
		ID: u.CallbackQuery.GetMsgID(),
		Message: i18n.T(i18nk.BotMsgUpdateInfoUpgradingWithVersion, map[string]any{
			"Current": config.Version,
		}),
	})
	latest, err := ghselfupdate.UpdateSelf(currentV, config.GitRepo)
	if err != nil {
		ctx.EditMessage(u.GetUserChat().GetID(), &tg.MessagesEditMessageRequest{
			ID: u.CallbackQuery.GetMsgID(),
			Message: i18n.T(i18nk.BotMsgUpdateErrorUpgradeFailed, map[string]any{
				"Error": err.Error(),
			}),
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(u.GetUserChat().GetID(), &tg.MessagesEditMessageRequest{
		ID: u.CallbackQuery.GetMsgID(),
		Message: i18n.T(i18nk.BotMsgUpdateInfoUpgradeSuccess, map[string]any{
			"Version": latest.Version.String(),
		}),
	})
	return errors.New("SAVEANTBOT-RESTART")
}
