package iiif

import ()

type Cache interface {
	Get(string) ([]byte, error)
	Set(string, []byte) error
	Unset(string) error
}

type Image interface {
	Body() ([]byte, error)
	Format() string
	ContentType() string
}

type Level interface {
}

type Profile interface {
}
