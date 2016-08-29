package cache

import (
	"github.com/thisisaaronland/iiif"
	"github.com/thisisaaronland/iiif/config"
)

func NewCacheFromConfig(cfg config.CacheConfig) (iiif.Cache, error) {

	if cfg.Name == "Disk" {
		cache, err := NewDiskCache(cfg)
		return cache, err
	} else if cfg.Name == "Memory" {
		cache, err := NewMemoryCache(cfg)
		return cache, err
	} else {
		cache, err := NewNullCache(cfg)
		return cache, err
	}
}
