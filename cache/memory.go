package cache

import (
	"errors"
	gocache "github.com/patrickmn/go-cache"
	"github.com/thisisaaronland/iiif"
	"time"
)

type MemoryCache struct {
	iiif.Cache
	cache gocache.Cache
}

func NewMemoryCache(config *iiif.Config) (*MemoryCache, error) {

	ttl := 5 * time.Minute // read from config
	flush := 30 * time.Second

	c := gocache.New(ttl, flush)

	mc := MemoryCache{
		cache: c,
	}

	return &mc, nil
}

func (mc *MemoryCache) Get(key string) ([]byte, error) {

	rsp, found := mc.cache.Get(key)

	if !found {
		err := errors.New("unknown key")
		return nil, err
	}

	return rsp.([]byte), err
}

func (mc *MemoryCache) Set(key string, body []byte) error {

	mc.cache.Set(key, body, gocache.DefaultExpiration)

	return nil
}

func (mc *MemoryCache) Unset(key string) error {

	mc.cache.Delete(key)
	return nil
}
