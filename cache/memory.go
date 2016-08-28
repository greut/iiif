package cache

import (
	"github.com/allegro/bigcache"
	"github.com/thisisaaronland/iiif"
	"time"
)

type MemoryCache struct {
	iiif.Cache
	cache *bigcache.BigCache
}

func NewMemoryCache(config iiif.CacheConfig) (*MemoryCache, error) {

	bconfig := bigcache.DefaultConfig(10 * time.Minute)
	bcache, err := bigcache.NewBigCache(bconfig)

	if err != nil {
		return nil, err
	}

	mc := MemoryCache{
		cache: bcache,
	}

	return &mc, nil
}

func (mc *MemoryCache) Get(key string) ([]byte, error) {

	rsp, err := mc.cache.Get(key)

	if err != nil {
		return nil, err
	}

	return rsp, nil
}

func (mc *MemoryCache) Set(key string, body []byte) error {

	mc.cache.Set(key, body)

	return nil
}

func (mc *MemoryCache) Unset(key string) error {

	return nil
}
