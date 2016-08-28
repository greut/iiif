package image

	// full
	// square
	// x,y,w,h (in pixels)
	// pct:x,y,w,h (in percents)

func IsValidRegion (region string) (bool, error) {
     return true, nil
}

	// max, full
	// w,h (deform)
	// !w,h (best fit within size)
	// w, (force width)
	// ,h (force height)
	// pct:n (resize)

func IsValidSize (size string) (bool, error) {
     return true, nil
}

	// n angle clockwise in degrees
	// !n angle clockwise in degrees with a flip (beforehand)

func IsValidRotation(rotation string) (bool,error) {
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
     Region string
     Size string
     Rotation string
     Quality string
     Format string
}

func NewTransformation (region string, size string, rotation string, quality string, format string) (*Transformation, error){

       ok bool
       err error

	  ok, err = image.IsValidRegion(region)

	  if !ok {
		return nil, err
	  }

	  ok, err := image.IsValidSize(size)

	  if !ok {
		return nil, err
	  }

	  ok, err := image.IsValidRotation(rotation)

	  if !ok {
		return nil, err
	  }

	  ok, err := image.IsValidQuality(quality)

	  if !ok {
		return nil, err
	  }

	  ok, err := image.IsValidFormat(format)

	  if !ok {
		return nil, err
	  }

	  t := Transformation{
	    Region: region,
	    Size: size,
	    Rotation: rotation,
	    Quality: quality,
	    Format: format,
	  }

	  return &t, nil
}