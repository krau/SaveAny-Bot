package common

import (
	"context"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	gocachestore "github.com/eko/gocache/store/go_cache/v4"
	gocache "github.com/patrickmn/go-cache"
)

var Cache *cache.Cache[any]

func initCache() {
	gocacheClient := gocache.New(time.Hour*1, time.Minute*10)
	gocacheStore := gocachestore.NewGoCache(gocacheClient)
	cacheManager := cache.New[any](gocacheStore)
	Cache = cacheManager
}

func CacheGet[T any](ctx context.Context, key string) (T, error) {
	data, err := Cache.Get(ctx, key)
	if err != nil {
		return *new(T), err
	}
	if v, ok := data.(T); ok {
		return v, nil
	}
	return *new(T), nil
}

func CacheSet(ctx context.Context, key string, value any) error {
	return Cache.Set(ctx, key, value)
}

func CacheDelete(ctx context.Context, key string) error {
	return Cache.Delete(ctx, key)
}
