package re

import "regexp"

var (
	TgMessageLinkRegexString = `https?://t\.me/(?:c/\d+|[A-Za-z0-9_]+)/\d+(?:/\d+)?(?:\?[^\s#]*[A-Za-z0-9_])?\b`
	TgMessageLinkRegexp      = regexp.MustCompile(TgMessageLinkRegexString)
	TelegraphUrlRegexString  = `https://telegra.ph/.*`
	TelegraphUrlRegexp       = regexp.MustCompile(TelegraphUrlRegexString)
)
