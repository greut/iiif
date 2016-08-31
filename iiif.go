package iiif

import ()

type Cache interface {
	Get(string) ([]byte, error)
	Set(string, []byte) error
	Unset(string) error
}

type Image interface {
	Identifier() string
	Body() ([]byte, error)
	Format() string
	ContentType() string
	Dimensions() Dimensions
}

type Dimensions interface {
	Height() int
	Width() int
}

type Level interface {
}

type Profile interface {
}
