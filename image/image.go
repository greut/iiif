package image

import ()

type Image interface {
	Identifier() string
	Height() int
	Width() int
	Transform(t *Transformation) ([]byte, error)
}
