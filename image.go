package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/golang/groupcache"
	"github.com/gorilla/mux"
	"gopkg.in/h2non/bimg.v1"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func resizeImage(vars map[string]string, cache *groupcache.Group) ([]byte, *time.Time, error) {
	identifier := vars["identifier"]
	format := vars["format"]

	identifier = strings.Replace(identifier, "../", "", -1)

	// Type
	bimgType := bimg.UNKNOWN
	if format == "jpg" || format == "jpeg" {
		bimgType = bimg.JPEG
	} else if format == "png" {
		bimgType = bimg.PNG
	} else if format == "webp" {
		bimgType = bimg.WEBP
	} else if format == "tif" || format == "tiff" {
		bimgType = bimg.TIFF
	} else if format == "gif" || format == "pdf" || format == "jp2" {
		message := fmt.Sprintf(formatMissing, format)
		return nil, nil, HTTPError{http.StatusNotImplemented, message}
	}

	if bimgType == bimg.UNKNOWN {
		message := fmt.Sprintf(formatError, format)
		return nil, nil, HTTPError{http.StatusBadRequest, message}
	}

	image, modTime, err := openImage(identifier, cache)
	if err != nil {
		return nil, nil, HTTPError{http.StatusNotFound, identifier}
	}

	size, err := image.Size()
	if err != nil {
		message := fmt.Sprintf(openError, err.Error())
		return nil, nil, HTTPError{http.StatusNotImplemented, message}
	}

	options := bimg.Options{
		Width:  size.Width,
		Height: size.Height,
		Type:   bimgType,
	}

	// Size & Region
	// ----
	// Bimg handles the zooming before the cropping
	err = handleSizeAndRegion(vars["size"], vars["region"], &options)
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

	quality := vars["quality"]
	format := vars["format"]

	images, _ := r.Context().Value(ContextKey("images")).(*groupcache.Group)
	thumbnails, _ := r.Context().Value(ContextKey("thumbnails")).(*groupcache.Group)

	sURL := r.URL.String()
	modTime := time.Now()

	var buffer []byte
	var err error
	if thumbnails != nil {
		var image = new(ImageWithModTime)
		err = thumbnails.Get(vars, sURL, groupcache.ProtoSink(image))
		buffer = image.GetBuffer()
		_ = modTime.UnmarshalBinary(image.GetModTime())
	} else {
		var mt *time.Time
		buffer, mt, err = resizeImage(vars, images)
		// On testing... mt might be null.
		if mt != nil {
			modTime = *mt
		}
	}

	if err != nil {
		e := err.(HTTPError)
		http.Error(w, e.Message, e.StatusCode)
		return
	}

	filename := fmt.Sprintf("%v.%v", quality, format)
	http.ServeContent(w, r, filename, modTime, bytes.NewReader(buffer))
}

func openImage(identifier string, cache *groupcache.Group) (*bimg.Image, *time.Time, error) {
	identifier, err := url.QueryUnescape(identifier)
	if err != nil {
		debug("Filename is frob %#v", identifier)
		return nil, nil, err
	}

	identifier = strings.Replace(identifier, "../", "", -1)

	filename := filepath.Join(*root, identifier)
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
				log.Printf("Not a base64 encoded URL either.")
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
			debug("Cannot open file %#v: %#v", filename, err.Error())
			return nil, nil, err
		}

	}

	modTime := time.Now()
	if stat != nil {
		modTime = stat.ModTime()
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
	defer resp.Body.Close()
	// XXX deal with last-modified-since...
	buf := bytes.NewBuffer(make([]byte, 0, resp.ContentLength))
	_, err = buf.ReadFrom(resp.Body)
	return buf.Bytes(), nil
}

func handleSizeAndRegion(size string, region string, opts *bimg.Options) error {
	// Size
	// ----
	// max, full
	// w,h (deform)
	// !w,h (best fit within size)
	// w, (force width)
	// ,h (force height)

	// Sizes
	var width int
	var height int
	var pct float64

	best := false

	if size != "max" && size != "full" {
		if strings.HasPrefix(size, "pct:") {
			var err error
			pct, err = strconv.ParseFloat(size[4:], 64)
			if err != nil || pct == 0 {
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
		// XXX Handle max w.r.t. the configuration (anti-DOS)
	}

	debug("Input sizes: %v x %v / %v (best: %v)", width, height, pct, best)

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
			opts.Gravity = bimg.GravitySmart
		}

		opts.Width = width
		opts.Height = height
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
	} else if quality == "gray" {
		opts.Interpretation = bimg.InterpretationGREY16
	} else if quality == "bitonal" {
		opts.Interpretation = bimg.InterpretationBW
	} else {
		message := fmt.Sprintf(qualityError, quality)
		return HTTPError{http.StatusBadRequest, message}
	}

	return nil
}
