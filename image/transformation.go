package image

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var regionError = "IIIF 2.1 `region` argument is not recognized: %#v"
var sizeError = "IIIF 2.1 `size` argument is not recognized: %#v"
var qualityError = "IIIF 2.1 `quality` and `format` arguments were expected: %#v"
var rotationError = "IIIF 2.1 `rotation` argument is not recognized: %#v"
var rotationMissing = "libvips cannot rotate angle that isn't a multiple of 90: %#v"

type RegionInstruction struct {
	X      int
	Y      int
	Height int
	Width  int
}

type SizeInstruction struct {
	Height  int
	Width   int
	Force   bool
	Enlarge bool
}

type RotationInstruction struct {
	Flip  bool
	Angle float64
}

// full
// square
// x,y,w,h (in pixels)
// pct:x,y,w,h (in percents)

func IsValidRegion(region string) (bool, error) {
	return true, nil
}

// max, full
// w,h (deform)
// !w,h (best fit within size)
// w, (force width)
// ,h (force height)
// pct:n (resize)

func IsValidSize(size string) (bool, error) {
	return true, nil
}

// n angle clockwise in degrees
// !n angle clockwise in degrees with a flip (beforehand)

func IsValidRotation(rotation string) (bool, error) {
	return true, nil
}

// color
// gray
// bitonal (not supported)
// default

func IsValidQuality(quality string) (bool, error) {
	return true, nil
}

func IsValidFormat(format string) (bool, error) {
	return true, nil
}

type Transformation struct {
	Region   string
	Size     string
	Rotation string
	Quality  string
	Format   string
}

func NewTransformation(region string, size string, rotation string, quality string, format string) (*Transformation, error) {

	var ok bool
	var err error

	ok, err = IsValidRegion(region)

	if !ok {
		return nil, err
	}

	ok, err = IsValidSize(size)

	if !ok {
		return nil, err
	}

	ok, err = IsValidRotation(rotation)

	if !ok {
		return nil, err
	}

	ok, err = IsValidQuality(quality)

	if !ok {
		return nil, err
	}

	ok, err = IsValidFormat(format)

	if !ok {
		return nil, err
	}

	t := Transformation{
		Region:   region,
		Size:     size,
		Rotation: rotation,
		Quality:  quality,
		Format:   format,
	}

	return &t, nil
}

func (t *Transformation) RegionInstructions(im *Image) (*RegionInstruction, error) {

	dims, err := im.Dimensions()

	if err != nil {
		return nil, err
	}

	width := dims.Width()
	height := dims.Height()

	if t.Region == "square" {

		var x int
		var y int

		if width < height {
			y = (height - width) / 2.
			x = width
		} else {
			x = (width - height) / 2.
			y = height
		}

		y = x

		instruction := RegionInstruction{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
		}

		return &instruction, nil
	}

	arr := strings.Split(t.Region, ":")

	if len(arr) == 1 {

		sizes := strings.Split(arr[0], ",")

		if len(sizes) != 4 {
			message := fmt.Sprintf("Invalid region")
			return nil, errors.New(message)
		}

		x, err := strconv.ParseInt(sizes[0], 10, 64)

		if err != nil {
			return nil, err
		}

		y, err := strconv.ParseInt(sizes[1], 10, 64)

		if err != nil {
			return nil, err
		}

		w, err := strconv.ParseInt(sizes[2], 10, 64)

		if err != nil {
			return nil, err
		}

		h, err := strconv.ParseInt(sizes[3], 10, 64)

		if err != nil {
			return nil, err
		}

		instruction := RegionInstruction{
			Width:  int(w),
			Height: int(h),
			X:      int(x),
			Y:      int(y),
		}

		return &instruction, nil

	}

	if arr[0] == "pct" {

		sizes := strings.Split(arr[1], ",")

		if len(sizes) != 4 {
			message := fmt.Sprintf("Invalid region", t.Region)
			return nil, errors.New(message)
		}

		x, err := strconv.ParseFloat(sizes[0], 64)

		if err != nil {
			return nil, err
		}

		y, err := strconv.ParseFloat(sizes[1], 64)

		if err != nil {
			return nil, err
		}

		w, err := strconv.ParseFloat(sizes[2], 64)

		if err != nil {
			return nil, err
		}

		h, err := strconv.ParseFloat(sizes[3], 64)

		if err != nil {
			return nil, err
		}

		w = int(math.Ceil(float64(width) * w / 100.))
		h = int(math.Ceil(float64(height) * h / 100.))
		x = int(math.Ceil(float64(width) * x / 100.))
		y = int(math.Ceil(float64(height) * y / 100.))

		instruction := RegionInstruction{
			Width:  w,
			Height: h,
			X:      x,
			Y:      y,
		}

		return &instruction, nil

	} else {
	}

	message := fmt.Sprintf("Unrecognized region")
	return nil, errors.New(message)

}

func (t *Transformation) SizeInstructions(im *Image) (*SizeInstruction, error) {

	w := 0
	h := 0
	force := false
	enlarge := false

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

			w = int(wi)
			h = int(h)

			if best {
				enlarge = true
			} else {
				force = true
			}

		} else if err_h != nil {
			w = int(wi)
			h = 0
		} else {
			w = 0
			h = int(h)
		}

		d := Dimensions{
			Height:  h,
			Width:   w,
			Enlarge: enlarge,
			Force:   force,
		}

		return &d, nil

	} else if arr[0] == "pct" {

		pct, err := strconv.ParseFloat(arr[1], 64)

		if err != nil {
			err := errors.New("invalid size")
			return nil, err
		}

		dims, err := im.Dimensions()

		if err != nil {
			return err
		}

		width := dims.Width()
		height := dims.Height()

		w = int(math.Ceil(pct / 100 * float64(width)))
		h = int(math.Ceil(pct / 100 * float64(height)))

	} else {

		message := fmt.Sprintf(sizeError, t.Size)
		return nil, errors.New(message)
	}

	instruction := SizeInstruction{
		Height:  h,
		Width:   w,
		Enlarge: enlarge,
		Force:   force,
	}

	return &instruction, nil

}

func (t *Transformation) RotationInstructions(im *Image) (*RotationInstruction, error) {

	flip := strings.HasPrefix(t.Rotation, "!")
	angle, err := strconv.ParseInt(strings.Trim(t.Rotation, "!"), 10, 64)

	if err != nil {
		message := fmt.Sprintf(rotationError, t.Rotation)
		return nil, errors.New(message)

	} else if angle%90 != 0 {
		message := fmt.Sprintf(rotationMissing, t.Rotation)
		return nil, errors.New(message)
	}

	instruction := RotationInstruction{
		Flip:  flip,
		Angle: angle,
	}

	return &instruction, nil
}
