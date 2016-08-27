package image

import (
	"gopkg.in/h2non/bimg.v1"
	"path/filepath"
)

type Derivative interface {
     Identifier() string
     Height() int
     Width() int
}

type Derivative struct {
     id string
     bimg bimg.Image
     cache iiif.Cache
}

function NewDerivative (bytes []byte) (*Derivative, error) {


}