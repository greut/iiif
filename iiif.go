package iiif

import ()

type Cache interface {
	Get(string) ([]byte, error)
	Set(string, []byte) error
	Unset(string) error
}

type Level interface {
}

type Profile interface {
}
