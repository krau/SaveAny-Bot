package cache

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/dgraph-io/ristretto/v2"
)

var cache *ristretto.Cache[string, any]


// TODO: maybe we should use simple ttl cache instead of ristretto...
func init() {
	c, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1e5,
		MaxCost:     1e6, // 1000000 / 112 ≈ 8928
		BufferItems: 64,
		OnReject: func(item *ristretto.Item[any]) {
			log.Warnf("Cache item rejected: key=%d, value=%v", item.Key, item.Value)
		},
	})
	if err != nil {
		log.Fatalf("failed to create ristretto cache: %v", err)
	}
	cache = c
}

func Set(key string, value any) error {
	ok := cache.Set(key, value, 0)
	if !ok {
		return fmt.Errorf("failed to set value in cache")
	}
	cache.Wait()
	return nil
}

func Get[T any](key string) (T, bool) {
	v, ok := cache.Get(key)
	if !ok {
		var zero T
		return zero, false
	}
	vT, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return vT, true
}
