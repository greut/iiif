package cache

import (
	"errors"
	"github.com/thisisaaronland/iiif"
	"github.com/thisisaaronland/iiif/config"
)

type NullCache struct {
	iiif.Cache
}

func NewNullCache(cfg config.CacheConfig) (*NullCache, error) {

	c := NullCache{}

	return &c, nil
}

func (c *NullCache) Get(rel_path string) ([]byte, error) {

	err := errors.New("null cache is null")
	return nil, err
}

func (c *NullCache) Set(rel_path string, body []byte) error {

	return nil
}

func (c *NullCache) Unset(rel_path string) error {

	return nil
}
