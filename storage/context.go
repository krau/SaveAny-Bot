package storage

import (
	"context"

	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
)

type contextKey struct{}

var storageKey = contextKey{}

func WithContext(ctx context.Context, storage Storage) context.Context {
	if storage == nil {
		return ctx
	}
	return context.WithValue(ctx, storageKey, storage)
}

func FromContext(ctx context.Context) Storage {
	storage, ok := ctx.Value(storageKey).(Storage)
	if !ok {
		return nil
	}
	return storage
}

func WithOverwrite(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxkey.OverwriteExisting, true)
}

func ShouldOverwrite(ctx context.Context) bool {
	overwrite, ok := ctx.Value(ctxkey.OverwriteExisting).(bool)
	return ok && overwrite
}
