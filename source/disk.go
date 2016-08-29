package source

import (
	"github.com/thisisaaronland/iiif"
	"github.com/thisisaaronland/iiif/cache"
	"github.com/thisisaaronland/iiif/config"
	"io/ioutil"
	"path/filepath"
)

type DiskSource struct {
	Source
	root  string
	cache iiif.Cache
}

func NewDiskSource(cfg config.ImagesConfig) (*DiskSource, error) {

	ch, err := cache.NewCacheFromConfig(cfg.Cache)

	if err != nil {
		return nil, err
	}

	ds := DiskSource{
		root:  cfg.Source.Path,
		cache: ch,
	}

	return &ds, nil
}

func (ds *DiskSource) Read(uri string) ([]byte, error) {

	body, err := ds.cache.Get(uri)

	if err == nil {
		return body, nil
	}

	abs_path := filepath.Join(ds.root, uri)

	body, err = ioutil.ReadFile(abs_path)

	if err != nil {
		return nil, err
	}

	go func() {
		ds.cache.Set(uri, body)
	}()

	return body, nil
}
