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

	// Region
	// ------
	err = handleRegion(vars["region"], &options)
	if err != nil {
		return nil, nil, err
	}

	// Size
	// ----
	err = handleSize(vars["size"], &options)
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
		log.Printf("Filename is frob %#v", identifier)
		return nil, nil, err
	}

	identifier = strings.Replace(identifier, "../", "", -1)

	filename := filepath.Join(*root, identifier)
	stat, err := os.Stat(filename)
	var buffer []byte
	if err != nil {
		log.Printf("Cannot open file %#v: %#v", filename, err.Error())
		url, err := base64.StdEncoding.DecodeString(identifier)
		if err != nil {
			log.Printf("Not a base64 encoded URL either.")
			return nil, nil, err
		}

		sURL := string(url)
		if cache != nil {
			err = cache.Get(nil, sURL, groupcache.AllocatingByteSliceSink(&buffer))
			if err != nil {
				return nil, nil, err
			}
			log.Printf("From cache %v\n", sURL)
		} else {
			buffer, err = downloadImage(sURL)
			if err != nil {
				return nil, nil, err
			}
		}
	} else {
		buffer, err = bimg.Read(filename)
		if err != nil {
			log.Printf("Cannot open file %#v: %#v", filename, err.Error())
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
	log.Printf("downloading %v\n", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Download error: %q : %#v.", url, err)
		return nil, err
	}
	defer resp.Body.Close()
	// XXX deal with last-modified-since...
	buf := bytes.NewBuffer(make([]byte, 0, resp.ContentLength))
	_, err = buf.ReadFrom(resp.Body)
	return buf.Bytes(), nil
}

func handleRegion(region string, opts *bimg.Options) error {
	// full
	// square
	// x,y,w,h (in pixels)
	// pct:x,y,w,h (in percents)
	// smart (extension)
	opts.AreaWidth = opts.Width
	opts.AreaHeight = opts.Height
	opts.Top = 0
	opts.Left = 0

	if region != "full" {
		if region == "square" {
			if opts.Width < opts.Height {
				opts.Height = opts.Width
			} else {
				opts.Width = opts.Height
			}
			opts.Crop = true
			opts.Force = false
			opts.Gravity = bimg.GravityCentre
		} else if region == "smart" {
			opts.Crop = true
			opts.Gravity = bimg.GravitySmart
			opts.SmartCrop = true
		} else {
			if !strings.HasPrefix(region, "pct:") {
				sizes := strings.Split(region, ",")
				if len(sizes) != 4 {
					message := fmt.Sprintf(regionError, region)
					return HTTPError{http.StatusBadRequest, message}
				}
				x, errX := strconv.ParseInt(sizes[0], 10, 64)
				y, errY := strconv.ParseInt(sizes[1], 10, 64)
				w, errW := strconv.ParseInt(sizes[2], 10, 64)
				h, errH := strconv.ParseInt(sizes[3], 10, 64)

				if errX != nil || errY != nil || errW != nil || errH != nil {
					message := fmt.Sprintf(regionError, region)
					return HTTPError{http.StatusBadRequest, message}
				}

				opts.AreaWidth = int(w)
				opts.AreaHeight = int(h)
				opts.Left = int(x)
				opts.Top = int(y)
			} else {
				sizes := strings.Split(region[4:], ",")
				if len(sizes) != 4 {
					message := fmt.Sprintf(regionError, region)
					return HTTPError{http.StatusBadRequest, message}
				}
				x, errX := strconv.ParseFloat(sizes[0], 64)
				y, errY := strconv.ParseFloat(sizes[1], 64)
				w, errW := strconv.ParseFloat(sizes[2], 64)
				h, errH := strconv.ParseFloat(sizes[3], 64)

				if errX != nil || errY != nil || errW != nil || errH != nil {
					message := fmt.Sprintf(regionError, region)
					return HTTPError{http.StatusBadRequest, message}
				}

				opts.AreaWidth = int(float64(opts.Width) * w / 100.)
				opts.AreaHeight = int(float64(opts.Height) * h / 100.)
				opts.Left = int(float64(opts.Width) * x / 100.)
				opts.Top = int(float64(opts.Height) * y / 100.)
			}
		}

		// Hack: libvips does strange things here.
		// * https://github.com/h2non/bimg/issues/60
		// * https://github.com/h2non/bimg/commit/b7eaa00f104a8eab49eedf49d75b11308df95f7a
		if opts.Top <= 0 && opts.Left == 0 {
			opts.Top = -1
		}
	}
	return nil
}

func handleSize(size string, opts *bimg.Options) error {
	// max, full
	// w,h (deform)
	// !w,h (best fit within size)
	// w, (force width)
	// ,h (force height)
	if size != "max" && size != "full" {
		opts.Crop = true
		if strings.HasPrefix(size, "pct:") {
			pct, err := strconv.ParseFloat(size[4:], 64)
			if err != nil {
				message := fmt.Sprintf(sizeError, size)
				return HTTPError{http.StatusBadRequest, message}
			}

			opts.Width = int(pct / 100 * float64(opts.Width))
			opts.Height = int(pct / 100 * float64(opts.Height))
		} else {
			best := strings.HasPrefix(size, "!")
			size = strings.Trim(size, "!")

			sizes := strings.Split(size, ",")

			wi, errW := strconv.ParseInt(sizes[0], 10, 64)
			h, errH := strconv.ParseInt(sizes[1], 10, 64)

			if errW != nil && errH != nil {
				message := fmt.Sprintf(sizeError, size)
				return HTTPError{http.StatusBadRequest, message}
			} else if errW == nil && errH == nil {
				opts.Width = int(wi)
				opts.Height = int(h)

				if best {
					inRatio := float64(opts.AreaWidth) / float64(opts.AreaHeight)
					outRatio := float64(opts.Width) / float64(opts.Height)
					if inRatio < outRatio {
						opts.Width = int(float64(opts.Width) * inRatio)
					} else {
						opts.Height = int(float64(opts.Height) / inRatio)
					}
				}
			} else if errW != nil {
				ratio := float64(opts.Height) / float64(opts.Width)
				opts.Width = int(h)
				opts.Height = int(float64(h) * ratio)
			} else {
				ratio := float64(opts.Width) / float64(opts.Height)
				opts.Height = int(wi)
				opts.Width = int(float64(wi) * ratio)
			}
		}
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
