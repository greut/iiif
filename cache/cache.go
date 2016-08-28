package cache

import (
	"github.com/thisisaaronland/iiif"
)

func NewCacheFromConfig(config iiif.CacheConfig) (iiif.Cache, error) {

	if config.Name == "Disk" {
		cache, err := NewDiskCache(config)
		return cache, err
	} else if config.Name == "Memory" {
		cache, err := NewMemoryCache(config)
		return cache, err
	} else {
		cache, err := NewNullCache(config)
		return cache, err
	}
}
