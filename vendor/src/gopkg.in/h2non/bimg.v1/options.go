package bimg

/*
#cgo pkg-config: vips
#include "vips/vips.h"
*/
import "C"

const (
	// Quality defines the default JPEG quality to be used.
	Quality = 80
	// MaxSize defines the maximum pixels width or height supported.
	MaxSize = 16383
)

// Gravity represents the image gravity value.
type Gravity int

const (
	// GravityCentre represents the centre value used for image gravity orientation.
	GravityCentre Gravity = iota
	// GravityNorth represents the north value used for image gravity orientation.
	GravityNorth
	// GravityEast represents the east value used for image gravity orientation.
	GravityEast
	// GravitySouth represents the south value used for image gravity orientation.
	GravitySouth
	// GravityWest represents the west value used for image gravity orientation.
	GravityWest
)

// Interpolator represents the image interpolation value.
type Interpolator int

const (
	// Bicubic interpolation value.
	Bicubic Interpolator = iota
	// Bilinear interpolation value.
	Bilinear
	// Nohalo interpolation value.
	Nohalo
)

var interpolations = map[Interpolator]string{
	Bicubic:  "bicubic",
	Bilinear: "bilinear",
	Nohalo:   "nohalo",
}

func (i Interpolator) String() string {
	return interpolations[i]
}

// Angle represents the image rotation angle value.
type Angle int

const (
	// D0 represents the rotation angle 0 degrees.
	D0 Angle = 0
	// D90 represents the rotation angle 90 degrees.
	D90 Angle = 90
	// D180 represents the rotation angle 180 degrees.
	D180 Angle = 180
	// D270 represents the rotation angle 270 degrees.
	D270 Angle = 270
)

// Direction represents the image direction value.
type Direction int

const (
	// Horizontal represents the orizontal image direction value.
	Horizontal Direction = C.VIPS_DIRECTION_HORIZONTAL
	// Vertical represents the vertical image direction value.
	Vertical Direction = C.VIPS_DIRECTION_VERTICAL
)

// Interpretation represents the image interpretation type.
// See: http://www.vips.ecs.soton.ac.uk/supported/current/doc/html/libvips/VipsImage.html#VipsInterpretation
type Interpretation int

const (
	// InterpretationError points to the libvips interpretation error type.
	InterpretationError Interpretation = C.VIPS_INTERPRETATION_ERROR
	// InterpretationMultiband points to its libvips interpretation equivalent type.
	InterpretationMultiband Interpretation = C.VIPS_INTERPRETATION_MULTIBAND
	// InterpretationBW points to its libvips interpretation equivalent type.
	InterpretationBW Interpretation = C.VIPS_INTERPRETATION_B_W
	// InterpretationCMYK points to its libvips interpretation equivalent type.
	InterpretationCMYK Interpretation = C.VIPS_INTERPRETATION_CMYK
	// InterpretationRGB points to its libvips interpretation equivalent type.
	InterpretationRGB Interpretation = C.VIPS_INTERPRETATION_RGB
	// InterpretationSRGB points to its libvips interpretation equivalent type.
	InterpretationSRGB Interpretation = C.VIPS_INTERPRETATION_sRGB
	// InterpretationRGB16 points to its libvips interpretation equivalent type.
	InterpretationRGB16 Interpretation = C.VIPS_INTERPRETATION_RGB16
	// InterpretationGREY16 points to its libvips interpretation equivalent type.
	InterpretationGREY16 Interpretation = C.VIPS_INTERPRETATION_GREY16
	// InterpretationScRGB points to its libvips interpretation equivalent type.
	InterpretationScRGB Interpretation = C.VIPS_INTERPRETATION_scRGB
	// InterpretationLAB points to its libvips interpretation equivalent type.
	InterpretationLAB Interpretation = C.VIPS_INTERPRETATION_LAB
	// InterpretationXYZ points to its libvips interpretation equivalent type.
	InterpretationXYZ Interpretation = C.VIPS_INTERPRETATION_XYZ
)

// WatermarkFont defines the default watermark font to be used.
var WatermarkFont = "sans 10"

// Color represents a traditional RGB color scheme.
type Color struct {
	R, G, B uint8
}

// ColorBlack is a shortcut to black RGB color representation.
var ColorBlack = Color{0, 0, 0}

// Watermark represents the text-based watermark supported options.
type Watermark struct {
	Width       int
	DPI         int
	Margin      int
	Opacity     float32
	NoReplicate bool
	Text        string
	Font        string
	Background  Color
}

// GaussianBlur represents the gaussian image transformation values.
type GaussianBlur struct {
	Sigma   float64
	MinAmpl float64
}

// Sharpen represents the image sharp transformation options.
type Sharpen struct {
	Radius int
	X1     float64
	Y2     float64
	Y3     float64
	M1     float64
	M2     float64
}

// Options represents the supported image transformation options.
type Options struct {
	Height         int
	Width          int
	AreaHeight     int
	AreaWidth      int
	Top            int
	Left           int
	Extend         int
	Quality        int
	Compression    int
	Zoom           int
	Crop           bool
	Enlarge        bool
	Embed          bool
	Flip           bool
	Flop           bool
	Force          bool
	NoAutoRotate   bool
	NoProfile      bool
	Interlace      bool
	Rotate         Angle
	Background     Color
	Gravity        Gravity
	Watermark      Watermark
	Type           ImageType
	Interpolator   Interpolator
	Interpretation Interpretation
	GaussianBlur   GaussianBlur
	Sharpen        Sharpen
}
