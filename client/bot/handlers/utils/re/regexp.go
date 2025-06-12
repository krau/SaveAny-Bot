package re

import "regexp"

var (
	TgMessageLinkRegexString = `https?://t\.me/(?:c/\d+|[a-zA-Z0-9_]+)/\d+(?:\?[^\s]*)?`
	TgMessageLinkRegexp       = regexp.MustCompile(TgMessageLinkRegexString)
	TelegraphUrlRegexString  = `https://telegra.ph/.*`
	TelegraphUrlRegexp        = regexp.MustCompile(TelegraphUrlRegexString)
)
