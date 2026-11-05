package fnamest

import "strings"

//go:generate go-enum --values --names --noprefix --flag --nocase

// FnameST
/* ENUM(
default, message, template
) */
type FnameST string

var fnameSTDisplay = map[FnameST]map[string]string{
	Default:  {"zh-Hans": "默认", "en": "Default"},
	Message:  {"zh-Hans": "优先从消息生成", "en": "Gen From Msg First"},
	Template: {"zh-Hans": "自定义模板", "en": "Template"},
}

func GetDisplay(st FnameST, lang string) string {
	display, ok := fnameSTDisplay[st]
	if !ok {
		return string(st)
	}
	if str, ok := display[lang]; ok {
		return str
	}
	if strings.HasPrefix(strings.ToLower(lang), "zh") {
		return display["zh-Hans"]
	}
	return display["en"]
}
