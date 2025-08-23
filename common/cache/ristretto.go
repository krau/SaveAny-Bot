package cache

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/krau/SaveAny-Bot/config"
)

var cache *ristretto.Cache[string, any]

func Init() {
	if cache != nil {
		panic("cache already initialized")
	}
	c, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: config.C().Cache.NumCounters,
		MaxCost:     config.C().Cache.MaxCost,
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
	ok := cache.SetWithTTL(key, value, 0, time.Duration(config.C().Cache.TTL)*time.Second)
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
