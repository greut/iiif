package image

// PLEASE TO RENAME THIS PACKAGE AND METHODS AS image/bimg.go

import (
	"errors"
	"fmt"
	"github.com/thisisaaronland/iiif"
	"github.com/thisisaaronland/iiif/source"
	"gopkg.in/h2non/bimg.v1"
)

type Image struct {
	iiif.Image
	source source.Source
	id     string
	bimg   *bimg.Image
}

type Dimensions struct {
	iiif.Dimensions
	imagesize bimg.ImageSize
}

func NewImageFromSource(src source.Source, id string) (*Image, error) {

	body, err := src.Read(id)

	if err != nil {
		return nil, err
	}

	bimg := bimg.NewImage(body)

	im := Image{
		source: src,
		id:     id,
		bimg:   bimg,
	}

	return &im, nil
}

func (im *Image) Body() []byte {
	return im.bimg.Image()
}

func (im *Image) Format() string {
	return im.bimg.Type()
}

func (im *Image) ContentType() string {

	format := im.Format()

	if format == "jpg" || format == "jpeg" {
		return "image/jpg"
	} else if format == "png" {
		return "image/png"
	} else if format == "webp" {
		return "image/webp"
	} else if format == "tif" || format == "tiff" {
		return "image/tiff"
	} else {
		return ""
	}
}

func (im *Image) Identifier() string {
	return im.id
}

func (im *Image) Dimensions() (*Dimensions, error) {

	sz, err := im.bimg.Size()

	if err != nil {
		return nil, err
	}

	d := Dimensions{
		imagesize: sz,
	}

	return &d, nil
}

func (im *Image) Transform(t *Transformation) error {

	var opts bimg.Options

	if t.Region != "full" {

		rgi, err := t.RegionInstructions(im)

		if err != nil {
			return err
		}

		opts = bimg.Options{
			AreaWidth:  rgi.Width,
			AreaHeight: rgi.Height,
			Left:       rgi.X,
			Top:        rgi.Y,
		}

		_, err = im.bimg.Process(opts)

		if err != nil {
			return err
		}
	}

	// QUESTION: after Process what are the dimensions of im.bimg ?

	dims, err := im.Dimensions()

	if err != nil {
		return err
	}

	opts = bimg.Options{
		Width:  dims.Width(),  // opts.AreaWidth,
		Height: dims.Height(), // opts.AreaHeight,
	}

	if t.Size != "max" && t.Size != "full" {

		si, err := t.SizeInstructions(im)

		if err != nil {
			return err
		}

		opts.Height = si.Height
		opts.Width = si.Width
		opts.Enlarge = si.Enlarge
		opts.Force = si.Force
	}

	ri, err := t.RotationInstructions(im)

	if err != nil {
		return nil
	}

	opts.Flip = ri.Flip
	opts.Rotate = bimg.Angle(ri.Angle % 360)

	if t.Quality == "color" || t.Quality == "default" {
		// do nothing.
	} else if t.Quality == "gray" {
		// FIXME: causes segmentation fault (core dumped)
		//options.Interpretation = bimg.InterpretationGREY16
		opts.Interpretation = bimg.InterpretationBW
	} else if t.Quality == "bitonal" {
		opts.Interpretation = bimg.InterpretationBW
	} else {
		message := fmt.Sprintf(qualityError, t.Quality)
		return errors.New(message)
	}

	_, err = im.bimg.Process(opts)

	if err != nil {
		return err
	}

	return nil
}

func (d *Dimensions) Height() int {
	return d.imagesize.Height
}

func (d *Dimensions) Width() int {
	return d.imagesize.Width
}
