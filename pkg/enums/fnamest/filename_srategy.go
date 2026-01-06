package fnamest

//go:generate go-enum --values --names --noprefix --flag --nocase

// FnameST
/* ENUM(
default, message, template
) */
type FnameST string

var fnameSTDisplay = map[FnameST]map[string]string{
	Default:  {"zh-CN": "默认", "en": "Default"},
	Message:  {"zh-CN": "优先从消息生成", "en": "Gen From Msg First"},
	Template: {"zh-CN": "自定义模板", "en": "Template"},
}

func GetDisplay(st FnameST, lang string) string {
	if display, ok := fnameSTDisplay[st]; ok {
		if str, ok := display[lang]; ok {
			return str
		}
	}
	return fnameSTDisplay[st]["en"]
}
