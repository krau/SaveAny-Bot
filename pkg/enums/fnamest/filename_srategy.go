package fnamest

//go:generate go-enum --values --names --noprefix --flag --nocase

// FnameST
/* ENUM(
default, message
) */
type FnameST string

var FnameSTDisplay = map[FnameST]string{
	Default: "默认",
	Message: "优先从消息生成",
}
