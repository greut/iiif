package source

import (
	"errors"
	"github.com/thisisaaronland/iiif"
)

type Source interface {
	Read(uri string) ([]byte, error)
}

func NewSourceFromConfig(config iiif.ImagesConfig) (Source, error) {

	if config.Source.Name == "Disk" {
		cache, err := NewDiskSource(config)
		return cache, err
	} else {
		err := errors.New("Unknown source type")
		return nil, err
	}
}
