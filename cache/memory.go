package cache

import (
	"github.com/allegro/bigcache"
	"github.com/thisisaaronland/iiif/config"
	"time"
)

type MemoryCache struct {
	Cache
	cache *bigcache.BigCache
}

func NewMemoryCache(cfg config.CacheConfig) (*MemoryCache, error) {

	ttl := cfg.TTL
	limit := cfg.Limit

	/*
	   ttl, err := strconv.Atoi(cfg.TTL)

	   if err != nil {
	   	   return nil, err
	   }

	   limit, err := strconv.Atoi(cfg.Limit)

	   if err != nil {
	   	   return nil, err
	   }
	*/

	window := time.Duration(ttl) * time.Second

	bconfig := bigcache.DefaultConfig(10 * time.Minute)
	bconfig.LifeWindow = window
	bconfig.HardMaxCacheSize = limit

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
