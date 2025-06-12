package re

import "regexp"

var (
	TgMessageLinkRegexString = `https?://t\.me/(?:c/\d+|[a-zA-Z0-9_]+)/\d+(?:\?[^\s]*)?`
	TgMessageLinkRegex       = regexp.MustCompile(TgMessageLinkRegexString)
)
