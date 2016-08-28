package image

import (
	"errors"
	"fmt"
	"github.com/thisisaaronland/iiif/source"
	"gopkg.in/h2non/bimg.v1"
	"math"
	"strconv"
	"strings"
)

var qualityError = "IIIF 2.1 `quality` and `format` arguments were expected: %#v"
var regionError = "IIIF 2.1 `region` argument is not recognized: %#v"
var sizeError = "IIIF 2.1 `size` argument is not recognized: %#v"
var rotationError = "IIIF 2.1 `rotation` argument is not recognized: %#v"
var rotationMissing = "libvips cannot rotate angle that isn't a multiple of 90: %#v"
var formatError = "IIIf 2.1 `format` argument is not yet recognized: %#v"
var formatMissing = "libvips cannot output this format %#v as of yet"

type Image struct {
	source source.Source
	id     string
	bimg   *bimg.Image
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

func (im *Image) Identifier() string {
	return im.id
}

func (im *Image) Height() int {
	size, _ := im.bimg.Size()
	return size.Height
}

func (im *Image) Width() int {
	size, _ := im.bimg.Size()
	return size.Width
}

func (im *Image) Transform(t *Transformation) ([]byte, error) {

	size, err := im.bimg.Size()

	if err != nil {
		return nil, err
	}

	if t.Region != "full" {

		w := size.Width
		h := size.Height

		opts := bimg.Options{
			AreaWidth:  w,
			AreaHeight: h,
			Top:        0,
			Left:       0,
		}

		if t.Region == "square" {

			if w < h {
				opts.Top = (h - w) / 2.
				opts.AreaWidth = w
			} else {
				opts.Left = (w - h) / 2.
				opts.AreaWidth = h
			}

			opts.AreaHeight = opts.AreaWidth

		} else {

			arr := strings.Split(t.Region, ":")

			if len(arr) == 1 {

				sizes := strings.Split(arr[0], ",")

				if len(sizes) != 4 {
					message := fmt.Sprintf(regionError, t.Region)
					return nil, errors.New(message)
				}

				x, _ := strconv.ParseInt(sizes[0], 10, 64)
				y, _ := strconv.ParseInt(sizes[1], 10, 64)
				w, _ := strconv.ParseInt(sizes[2], 10, 64)
				h, _ := strconv.ParseInt(sizes[3], 10, 64)

				opts.AreaWidth = int(w)
				opts.AreaHeight = int(h)
				opts.Left = int(x)
				opts.Top = int(y)

			} else if arr[0] == "pct" {

				sizes := strings.Split(arr[1], ",")

				if len(sizes) != 4 {
					message := fmt.Sprintf(regionError, t.Region)
					return nil, errors.New(message)
				}

				x, _ := strconv.ParseFloat(sizes[0], 64)
				y, _ := strconv.ParseFloat(sizes[1], 64)
				w, _ := strconv.ParseFloat(sizes[2], 64)
				h, _ := strconv.ParseFloat(sizes[3], 64)

				opts.AreaWidth = int(math.Ceil(float64(size.Width) * w / 100.))
				opts.AreaHeight = int(math.Ceil(float64(size.Height) * h / 100.))
				opts.Left = int(math.Ceil(float64(size.Width) * x / 100.))
				opts.Top = int(math.Ceil(float64(size.Height) * y / 100.))

			} else {
				message := fmt.Sprintf(regionError, t.Region)
				return nil, errors.New(message)
			}
		}

		_, err := im.bimg.Process(opts)

		if err != nil {
			return nil, err
		}

		size = bimg.ImageSize{
			Width:  opts.AreaWidth,
			Height: opts.AreaHeight,
		}

		// Size, Rotation and Quality are made in a single Process call.

		options := bimg.Options{
			Width:  size.Width,
			Height: size.Height,
		}

		if t.Size != "max" && t.Size != "full" {

			arr := strings.Split(t.Size, ":")

			if len(arr) == 1 {
				best := strings.HasPrefix(t.Size, "!")
				sizes := strings.Split(strings.Trim(arr[0], "!"), ",")

				if len(sizes) != 2 {
					message := fmt.Sprintf(sizeError, t.Size)
					return nil, errors.New(message)
				}

				wi, err_w := strconv.ParseInt(sizes[0], 10, 64)
				h, err_h := strconv.ParseInt(sizes[1], 10, 64)

				if err_w != nil && err_h != nil {
					message := fmt.Sprintf(sizeError, t.Size)
					return nil, errors.New(message)

				} else if err_w == nil && err_h == nil {
					options.Width = int(wi)
					options.Height = int(h)
					if best {
						options.Enlarge = true
					} else {
						options.Force = true
					}
				} else if err_h != nil {
					options.Width = int(wi)
					options.Height = 0
				} else {
					options.Width = 0
					options.Height = int(h)
				}
			} else if arr[0] == "pct" {
				pct, _ := strconv.ParseFloat(arr[1], 64)
				options.Width = int(math.Ceil(pct / 100 * float64(size.Width)))
				options.Height = int(math.Ceil(pct / 100 * float64(size.Height)))
			} else {
				message := fmt.Sprintf(sizeError, t.Size)
				return nil, errors.New(message)
			}
		}

		flip := strings.HasPrefix(t.Rotation, "!")
		angle, err := strconv.ParseInt(strings.Trim(t.Rotation, "!"), 10, 64)

		if err != nil {
			message := fmt.Sprintf(rotationError, t.Rotation)
			return nil, errors.New(message)

		} else if angle%90 != 0 {
			message := fmt.Sprintf(rotationMissing, t.Rotation)
			return nil, errors.New(message)
		}

		options.Flip = flip
		options.Rotate = bimg.Angle(angle % 360)

		if t.Quality == "color" || t.Quality == "default" {
			// do nothing.
		} else if t.Quality == "gray" {
			// FIXME: causes segmentation fault (core dumped)
			//options.Interpretation = bimg.InterpretationGREY16
			options.Interpretation = bimg.InterpretationBW
		} else if t.Quality == "bitonal" {
			options.Interpretation = bimg.InterpretationBW
		} else {
			message := fmt.Sprintf(qualityError, t.Quality)
			return nil, errors.New(message)
		}

		_, err = im.bimg.Process(options)

		if err != nil {
			return nil, err
		}
	}

	bytes := im.bimg.Image()
	return bytes, nil
}
