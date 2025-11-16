package dirutil

import (
	"context"

	"github.com/krau/SaveAny-Bot/database"
)

type contextKey struct{}

var dirContextKey = contextKey{}

func WithContext(ctx context.Context, dir *database.Dir) context.Context {
	if dir == nil {
		return ctx
	}
	return context.WithValue(ctx, dirContextKey, dir)
}

func FromContext(ctx context.Context) *database.Dir {
	dir, ok := ctx.Value(dirContextKey).(*database.Dir)
	if !ok {
		return nil
	}
	return dir
}

// PathFromContext returns the directory path stored in the context.
//
// If no directory is found, an empty string is returned.
func PathFromContext(ctx context.Context) string {
	dir := FromContext(ctx)
	if dir == nil {
		return ""
	}
	return dir.Path
}
