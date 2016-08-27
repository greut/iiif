package image

import (
	"gopkg.in/h2non/bimg.v1"
	"path/filepath"
)

type Image interface {
     Identifier() string
     Height() int
     Width() int
}

type Image struct {
     source string
     id string
     bimg bimg.Image
}

function NewImage (source string, id string) (*Image, error) {

	 filename := filepath.Join(source, id)

	 buffer, err := bimg.Read(filename)

	 if err != nil {
	    return nil, error
	 }

	 bimg := bimg.NewImage(buffer)

	 image := Image{
	 	source: source,
		id: id,	      
	        bimg: bimg,
	 }

	 return &image, nil
	 size, err := image.Size()
}

func (im *Image) Identifier() string {
     return im.id
}

func (im *Image) Height (int){
	size, _ := im.bimg.Size()
	return size.Height
}

func (im *Image) Width (int){
	size, _ := im.bimg.Size()
	return size.Width
}