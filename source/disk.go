package source

import (
	"github.com/thisisaaronland/iiif"
	"github.com/thisisaaronland/iiif/cache"
	"io/ioutil"
	"path/filepath"
)

type DiskSource struct {
	Source
	root  string
	cache iiif.Cache
}

func NewDiskSource(config iiif.ImagesConfig) (*DiskSource, error) {

	ch, err := cache.NewCacheFromConfig(config.Cache)

	if err != nil {
		return nil, err
	}

	ds := DiskSource{
		root:  config.Source.Path,
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
