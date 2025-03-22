package common

import (
	"bytes"
	"encoding/gob"
	"sync"

	"github.com/coocood/freecache"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/types"
)

type CommonCache struct {
	cache *freecache.Cache
	mu    sync.RWMutex
}

var Cache *CommonCache

func initCache() {
	gob.Register(types.File{})
	gob.Register(tg.InputDocumentFileLocation{})
	gob.Register(tg.InputPhotoFileLocation{})
	Cache = &CommonCache{cache: freecache.NewCache(10 * 1024 * 1024)}
}

func (c *CommonCache) Get(key string, value any) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, err := Cache.cache.Get([]byte(key))
	if err != nil {
		return err
	}
	dec := gob.NewDecoder(bytes.NewReader(data))
	err = dec.Decode(&value)
	if err != nil {
		return err
	}
	return nil
}

func (c *CommonCache) Set(key string, value any, expireSeconds int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(value)
	if err != nil {
		return err
	}
	Cache.cache.Set([]byte(key), buf.Bytes(), expireSeconds)
	return nil
}

func (c *CommonCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	Cache.cache.Del([]byte(key))
	return nil
}
