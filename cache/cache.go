package cache

import (
	"errors"
	"github.com/thisisaaronland/iiif"
)

func NewCacheFromConfig(config *iiif.Config) (iiif.Cache, error) {

	if config.Cache.Name == "Disk" {
		cache, err := NewDiskCache(config)
		return cache, err
	}

	cache, err := NewNullCache(config)
	return cache, err
}
