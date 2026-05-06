package ctxkey

// ENUM(content-length, overwrite-existing)
//
//go:generate go-enum --values --names --flag --nocase --noprefix
type ContextKey string
