package storage

import "context"

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
