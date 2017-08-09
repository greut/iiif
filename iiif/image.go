package iiif

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/golang/groupcache"
	"github.com/gorilla/mux"
	"gopkg.in/h2non/bimg.v1"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// error messages
var qualityError = "IIIF 2.1 `quality` and `format` arguments were expected: %#v"
var regionError = "IIIF 2.1 `region` argument is not recognized: %#v"
var sizeError = "IIIF 2.1 `size` argument is not recognized: %#v"
var maxSizeError = "The given `size` is out of the limits %vx%v (%vx%v or area %v)"
var rotationError = "IIIF 2.1 `rotation` argument is not recognized: %#v"
var rotationMissing = "libvips cannot rotate angle that isn't a multiple of 90: %#v"
var formatError = "IIIF 2.1 `format` argument is not yet recognized: %#v"
var formatMissing = "libvips cannot output this format %#v as of yet"
var formatReadMissing = "libvips cannot read this format %#v as of yet"

func resizeImage(config *Config, vars map[string]string, cache *groupcache.Group) ([]byte, *time.Time, error) {
	identifier := vars["identifier"]
	format := vars["format"]

	identifier = strings.Replace(identifier, "../", "", -1)

	// Type
	var bimgType bimg.ImageType
	if format == "jpg" {
		format = "jpeg"
	} else if format == "tif" {
		format = "tiff"
	} else if format == "jp2" {
		format = "magick"
	}

	for k, v := range bimg.ImageTypes {
		if v == format {
			bimgType = k
			break
		}
	}

	if !bimg.IsTypeSupportedSave(bimgType) {
		message := fmt.Sprintf(formatMissing, format)
		return nil, nil, HTTPError{http.StatusNotImplemented, message}
	}

	// Open image
	image, modTime, err := openImage(identifier, config.Images, cache)
	if err != nil {
		e, ok := err.(HTTPError)
		if ok {
			return nil, nil, e
		}
		return nil, nil, HTTPError{http.StatusNotFound, identifier}
	}

	size, err := image.Size()
	if err != nil {
		message := fmt.Sprintf(openError, err.Error())
		return nil, nil, HTTPError{http.StatusBadRequest, message}
	}

	options := bimg.Options{
		Width:  size.Width,
		Height: size.Height,
		Type:   bimgType,
	}

	// Size & Region
	// ----
	// Bimg handles the zooming before the cropping
	err = handleSizeAndRegion(vars["size"], vars["region"], config, &options)
	if err != nil {
		return nil, nil, err
	}

	// Quality
	// -------
	err = handleQuality(vars["quality"], &options)
	if err != nil {
		return nil, nil, err
	}

	_, err = image.Process(options)
	if err != nil {
		message := fmt.Sprintf("bimg couldn't process the image: %#v", err.Error())
		return nil, nil, HTTPError{http.StatusInternalServerError, message}
	}

	// Rotation
	// --------
	// n angle clockwise in degrees
	// !n angle clockwise in degrees with a flip (beforehand)
	rotation := vars["rotation"]
	flip := strings.HasPrefix(rotation, "!")
	angle, err := strconv.ParseInt(strings.Trim(rotation, "!"), 10, 64)
	angle %= 360

	if err != nil {
		message := fmt.Sprintf(rotationError, rotation)
		return nil, nil, HTTPError{http.StatusBadRequest, message}
	} else if angle%90 != 0 {
		message := fmt.Sprintf(rotationMissing, rotation)
		return nil, nil, HTTPError{http.StatusNotImplemented, message}
	}

	if flip || angle != 0 {
		options = bimg.Options{
			Flip:   flip,
			Rotate: bimg.Angle(angle),
		}
		_, err = image.Process(options)
		if err != nil {
			message := fmt.Sprintf("bimg couldn't process the image: %#v", err.Error())
			return nil, nil, HTTPError{http.StatusInternalServerError, message}
		}
	}

	return image.Image(), modTime, nil
}

// ImageHandler responds to the IIIF 2.1 Image API.
func ImageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	identifier := vars["identifier"]
	region := vars["region"]
	size := vars["size"]
	rotation := vars["rotation"]
	quality := vars["quality"]
	format := vars["format"]

	config, _ := r.Context().Value(ContextKey("config")).(*Config)
	images, _ := r.Context().Value(ContextKey("images")).(*groupcache.Group)
	thumbnails, _ := r.Context().Value(ContextKey("thumbnails")).(*groupcache.Group)

	sURL := r.URL.String()
	modTime := time.Now()

	var buffer []byte
	var err error
	if thumbnails != nil {
		var image = new(ImageWithModTime)
		ctx := struct {
			vars   map[string]string
			config *Config
		}{
			vars,
			config,
		}
		err = thumbnails.Get(ctx, sURL, groupcache.ProtoSink(image))
		buffer = image.GetBuffer()
		_ = modTime.UnmarshalBinary(image.GetModTime())
	} else {
		var mt *time.Time
		buffer, mt, err = resizeImage(config, vars, images)
		// When testing... mt might be null.
		if mt != nil {
			modTime = *mt
		}
	}

	if err != nil {
		e := err.(HTTPError)
		http.Error(w, e.Error(), e.StatusCode)
		return
	}

	filename := fmt.Sprintf("%v-%v-%v-%v-%v.%v", identifier, region, size, rotation, quality, format)
	filename = strings.Replace(
		strings.Replace(
			strings.Replace(filename, "/", "_", -1),
			":", "_", -1),
		",", "", -1)

	disposition := "inline"
	_, present := r.URL.Query()["dl"]
	if present {
		disposition = "attachement"
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, filename))

	http.ServeContent(w, r, filename, modTime, bytes.NewReader(buffer))
}

func openImage(identifier string, root string, cache *groupcache.Group) (*bimg.Image, *time.Time, error) {
	identifier, err := url.QueryUnescape(identifier)
	if err != nil {
		debug("Filename is frob %#v", identifier)
		return nil, nil, err
	}

	identifier = strings.Replace(identifier, "../", "", -1)

	filename := filepath.Join(root, identifier)
	stat, err := os.Stat(filename)
	var buffer []byte
	if err != nil {
		debug("Cannot open file %#v: %#v", filename, err.Error())
		var sURL string
		if strings.HasPrefix(identifier, "http:/") || strings.HasPrefix(identifier, "https:/") {
			sURL = strings.Replace(identifier, ":/", "://", 1)
		} else {
			url, err := base64.StdEncoding.DecodeString(identifier)
			if err != nil {
				debug("Not a base64 encoded URL either.")
				return nil, nil, err
			}
			sURL = string(url)
		}

		if cache != nil {
			err = cache.Get(nil, sURL, groupcache.AllocatingByteSliceSink(&buffer))
			if err != nil {
				return nil, nil, err
			}
			debug("From cache %v", sURL)
		} else {
			buffer, err = downloadImage(sURL)
			if err != nil {
				return nil, nil, err
			}
		}
	} else {
		buffer, err = bimg.Read(filename)
		if err != nil {
			return nil, nil, HTTPError{http.StatusBadRequest, err.Error()}
		}

	}

	modTime := time.Now()
	if stat != nil {
		modTime = stat.ModTime()
	}

	imageType := bimg.DetermineImageType(buffer)
	if !bimg.IsTypeSupported(imageType) {
		message := fmt.Sprintf(formatReadMissing, bimg.ImageTypes[imageType])
		return nil, nil, HTTPError{http.StatusNotImplemented, message}
	}

	image := bimg.NewImage(buffer)
	return image, &modTime, nil
}

func downloadImage(url string) ([]byte, error) {
	debug("downloading %v\n", url)

	resp, err := http.Get(url)
	if err != nil {
		debug("Download error: %q : %#v.", url, err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, HTTPError{resp.StatusCode, url}
	}

	defer resp.Body.Close()
	// XXX deal with last-modified-since...
	var buf []byte
	if resp.ContentLength > 0 {
		b := bytes.NewBuffer(make([]byte, 0, resp.ContentLength))
		_, err = b.ReadFrom(resp.Body)
		buf = b.Bytes()
	} else {
		buf, err = ioutil.ReadAll(resp.Body)
	}
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func handleSizeAndRegion(size string, region string, config *Config, opts *bimg.Options) error {
	// Size
	// ----
	// max, full
	// w,h (deform)
	// !w,h (best fit within size)
	// w, (force width)
	// ,h (force height)
	// pct: (scale the image of the extracted region in %)

	// Sizes
	var width int
	var height int
	var pct float64

	best := false
	isMax := false

	if size == "max" || size == "full" {
		isMax = config.MaxArea != 0 || config.MaxWidth != 0
	} else {
		if strings.HasPrefix(size, "pct:") {
			var err error
			pct, err = strconv.ParseFloat(size[4:], 64)
			if err != nil || pct <= 0 {
				message := fmt.Sprintf(sizeError, size)
				return HTTPError{http.StatusBadRequest, message}
			}
		} else {
			best = strings.HasPrefix(size, "!")
			sizes := strings.Split(strings.Trim(size, "!"), ",")

			if len(sizes) != 2 {
				message := fmt.Sprintf(sizeError, size)
				return HTTPError{http.StatusBadRequest, message}
			}

			w, errW := strconv.ParseInt(sizes[0], 10, 64)
			h, errH := strconv.ParseInt(sizes[1], 10, 64)

			if errW != nil && errH != nil || (w == 0 && h == 0) {
				message := fmt.Sprintf(sizeError, size)
				return HTTPError{http.StatusBadRequest, message}
			} else if errW == nil && errH == nil {
				width = int(w)
				height = int(h)

				if best {
					opts.Enlarge = true
				} else {
					opts.Force = true
				}
			} else if errH != nil {
				width = int(w)
			} else {
				height = int(h)
			}
		}

		if (config.MaxWidth != 0 && (config.MaxWidth < width || config.MaxHeight < height)) ||
			(config.MaxArea != 0 && config.MaxArea < width*height) {
			message := fmt.Sprintf(maxSizeError, width, height, config.MaxWidth, config.MaxHeight, config.MaxArea)
			return HTTPError{http.StatusBadRequest, message}
		}
	}

	debug("Input sizes: %v x %v / %v (best: %v, max: %v)", width, height, pct, best, isMax)

	// Region
	// ------
	// full
	// square
	// x,y,w,h (in pixels)
	// pct:x,y,w,h (in percents)
	// smart (extension)
	if region == "full" {
		if pct != 0 {
			opts.Width = int(float64(opts.Width) / 100. * pct)
			opts.Height = int(float64(opts.Height) / 100. * pct)
			debug("pct %v x %v", opts.Width, opts.Height)
		} else if width != 0 || height != 0 {
			opts.Width = width
			opts.Height = height
		} else {
			// The region is full and the size is max'd
			newW, newH := computeSize(opts.Width, opts.Height, config)
			opts.Width = newW
			opts.Height = newH
		}
	} else if region == "square" || region == "smart" {
		if width == 0 && height == 0 {
			width = opts.Width
			height = opts.Height
		}

		if region == "square" {
			if height == 0 {
				height = width
			} else if width == 0 {
				width = height
			} else {
				if width < height {
					height = width
				} else {
					width = height
				}
			}
			opts.Gravity = bimg.GravityCentre
		} else {
			// smart...
			r := float64(opts.Width) / float64(opts.Height)
			if height == 0 {
				height = int(float64(width) / r)
			} else if width == 0 {
				width = int(float64(height) * r)
			}
			opts.Gravity = bimg.GravitySmart
		}

		newW, newH := computeSize(width, height, config)

		opts.Width = newW
		opts.Height = newH
		opts.Crop = true
		opts.Force = false
	} else {
		isPercent := strings.HasPrefix(region, "pct:")
		var sizes []string
		if isPercent {
			sizes = strings.Split(region[4:], ",")
		} else {
			sizes = strings.Split(region, ",")
		}

		if len(sizes) != 4 {
			message := fmt.Sprintf(regionError, region)
			return HTTPError{http.StatusBadRequest, message}
		}

		var x, y, w, h int64
		var errX, errY, errW, errH error

		if isPercent {
			var xf, yf, wf, hf float64

			xf, errX = strconv.ParseFloat(sizes[0], 64)
			yf, errY = strconv.ParseFloat(sizes[1], 64)
			wf, errW = strconv.ParseFloat(sizes[2], 64)
			hf, errH = strconv.ParseFloat(sizes[3], 64)

			x = int64(float64(opts.Width) * xf / 100.)
			y = int64(float64(opts.Height) * yf / 100.)
			w = int64(float64(opts.Width) * wf / 100.)
			h = int64(float64(opts.Height) * hf / 100.)
		} else {
			x, errX = strconv.ParseInt(sizes[0], 10, 64)
			y, errY = strconv.ParseInt(sizes[1], 10, 64)
			w, errW = strconv.ParseInt(sizes[2], 10, 64)
			h, errH = strconv.ParseInt(sizes[3], 10, 64)
		}

		debug("Crop area: %v; %v (%v x %v)", x, y, w, h)

		if errX != nil || errY != nil || errW != nil || errH != nil ||
			x < 0 || y < 0 || w <= 0 || h <= 0 ||
			int(x+w) > opts.Width || int(y+h) > opts.Height {
			message := fmt.Sprintf(regionError, region)
			return HTTPError{http.StatusBadRequest, message}
		}

		if width == 0 || height == 0 {
			if width == 0 && height == 0 {
				if pct == 0 {
					width = int(w)
					height = int(h)
				} else {
					width = int(float64(w) / 100 * pct)
					height = int(float64(h) / 100 * pct)
				}
			} else {
				r := float64(w) / float64(h)
				if width == 0 {
					width = int(float64(height) * r)
				} else {
					height = int(float64(width) / r)
				}
			}
		}

		debug("Output size : %v x %v", width, height)

		// Calculate the new width/height...
		rW := float64(w) / float64(width)
		rH := float64(h) / float64(height)

		opts.AreaWidth = width
		opts.AreaHeight = height

		opts.Left = int(float64(x) / rW)
		opts.Top = int(float64(y) / rH)

		opts.Width = int(float64(opts.Width) / rW)
		opts.Height = int(float64(opts.Height) / rH)
	}

	return nil
}

func handleQuality(quality string, opts *bimg.Options) error {
	// color
	// gray
	// bitonal (not supported)
	// default
	// native (IIIF 1.0)
	if quality == "color" || quality == "default" || quality == "native" {
		// do nothing.
		return nil
	} else if quality == "gray" {
		opts.Interpretation = bimg.InterpretationGREY16
		return nil
	} else if quality == "bitonal" {
		message := fmt.Sprintf(qualityError, quality)
		return HTTPError{http.StatusNotImplemented, message}
	}

	message := fmt.Sprintf(qualityError, quality)
	return HTTPError{http.StatusBadRequest, message}
}

func computeSize(width, height int, config *Config) (int, int) {
	// The three ratios computed for each max value.
	rW := 1.
	rH := 1.
	rA := 1.

	if config.MaxWidth != 0 && width > config.MaxWidth {
		rW = float64(config.MaxWidth) / float64(width)
	}

	if config.MaxHeight != 0 && height > config.MaxHeight {
		rH = float64(config.MaxHeight) / float64(height)
	}

	area := width * height
	if config.MaxArea != 0 && area > config.MaxArea {
		rA = math.Sqrt(float64(config.MaxArea) / float64(area))
	}

	// Picking the smallest ratio enforces the smallest limitation
	ratio := math.Min(math.Min(rW, rH), rA)

	w := int(float64(width) * ratio)
	h := int(float64(height) * ratio)

	return w, h
}
