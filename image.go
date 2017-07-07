package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/golang/groupcache"
	"github.com/gorilla/mux"
	"gopkg.in/h2non/bimg.v1"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func resizeImage(vars map[string]string, cache *groupcache.Group) ([]byte, *time.Time, error) {
	quality := vars["quality"]
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

	// Region
	// ------
	opts, err := handleRegion(vars["region"], size, bimgType)
	if err != nil {
		return nil, nil, err
	}

	if opts != nil {
		// Hack: libvips does strange things here.
		// * https://github.com/h2non/bimg/issues/60
		// * https://github.com/h2non/bimg/commit/b7eaa00f104a8eab49eedf49d75b11308df95f7a
		if opts.Top <= 0 && opts.Left == 0 {
			opts.Top = -1
		}

		_, err = image.Process(*opts)
		if err != nil {
			return nil, nil, HTTPError{http.StatusInternalServerError, err.Error()}
		}

		size = bimg.ImageSize{
			Width:  opts.AreaWidth,
			Height: opts.AreaHeight,
		}
	}

	// Size, Rotation and Quality are made in a single Process call.
	options := bimg.Options{
		Width:  size.Width,
		Height: size.Height,
		Type:   bimgType,
	}

	// Size
	// ----
	// max, full
	// w,h (deform)
	// !w,h (best fit within size)
	// w, (force width)
	// ,h (force height)
	// pct:n (resize)
	s := vars["size"]
	if s != "max" && s != "full" {
		arr := strings.Split(s, ":")
		if len(arr) == 1 {
			best := strings.HasPrefix(s, "!")
			sizes := strings.Split(strings.Trim(arr[0], "!"), ",")

			if len(sizes) != 2 {
				message := fmt.Sprintf(sizeError, s)
				return nil, nil, HTTPError{http.StatusBadRequest, message}
			}

			wi, errW := strconv.ParseInt(sizes[0], 10, 64)
			h, errH := strconv.ParseInt(sizes[1], 10, 64)

			if errW != nil && errH != nil {
				message := fmt.Sprintf(sizeError, s)
				return nil, nil, HTTPError{http.StatusBadRequest, message}
			} else if errW == nil && errH == nil {
				options.Width = int(wi)
				options.Height = int(h)
				if best {
					options.Enlarge = true
				} else {
					options.Force = true
				}
			} else if errH != nil {
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
			message := fmt.Sprintf(sizeError, s)
			return nil, nil, HTTPError{http.StatusBadRequest, message}
		}
	}

	// Rotation
	// --------
	// n angle clockwise in degrees
	// !n angle clockwise in degrees with a flip (beforehand)
	rotation := vars["rotation"]
	flip := strings.HasPrefix(rotation, "!")
	angle, err := strconv.ParseInt(strings.Trim(rotation, "!"), 10, 64)

	if err != nil {
		message := fmt.Sprintf(rotationError, rotation)
		return nil, nil, HTTPError{http.StatusBadRequest, message}
	} else if angle%90 != 0 {
		message := fmt.Sprintf(rotationMissing, rotation)
		return nil, nil, HTTPError{http.StatusNotImplemented, message}
	}

	options.Flip = flip
	options.Rotate = bimg.Angle(angle % 360)

	// Quality
	// -------
	// color
	// gray
	// bitonal (not supported)
	// default
	// native (IIIF 1.0)
	if quality == "color" || quality == "default" || quality == "native" {
		// do nothing.
	} else if quality == "gray" {
		options.Interpretation = bimg.InterpretationGREY16
	} else if quality == "bitonal" {
		options.Interpretation = bimg.InterpretationBW
	} else {
		message := fmt.Sprintf(qualityError, quality)
		return nil, nil, HTTPError{http.StatusBadRequest, message}
	}

	_, err = image.Process(options)
	if err != nil {
		message := fmt.Sprintf("bimg couldn't process the image: %#v", err.Error())
		return nil, nil, HTTPError{http.StatusInternalServerError, message}
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
		modTime = *mt
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

func handleRegion(region string, size bimg.ImageSize, bimgType bimg.ImageType) (*bimg.Options, error) {
	// full
	// square
	// x,y,w,h (in pixels)
	// pct:x,y,w,h (in percents)
	if region != "full" {
		opts := bimg.Options{
			AreaWidth:  size.Width,
			AreaHeight: size.Height,
			Top:        0,
			Left:       0,
			Type:       bimgType,
		}
		if region == "square" {
			if size.Width < size.Height {
				opts.Top = (size.Height - size.Width) / 2.
				opts.AreaWidth = size.Width
			} else {
				opts.Left = (size.Width - size.Height) / 2.
				opts.AreaWidth = size.Height
			}
			opts.AreaHeight = opts.AreaWidth
		} else {
			arr := strings.Split(region, ":")
			if len(arr) == 1 {
				sizes := strings.Split(arr[0], ",")
				if len(sizes) != 4 {
					message := fmt.Sprintf(regionError, region)
					return nil, HTTPError{http.StatusBadRequest, message}
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
					message := fmt.Sprintf(regionError, region)
					return nil, HTTPError{http.StatusBadRequest, message}
				}
				x, _ := strconv.ParseFloat(sizes[0], 64)
				y, _ := strconv.ParseFloat(sizes[1], 64)
				w, _ := strconv.ParseFloat(sizes[2], 64)
				h, _ := strconv.ParseFloat(sizes[3], 64)
				opts.AreaWidth = int(math.Ceil(float64(size.Width) * w / 100.))
				opts.AreaHeight = int(math.Ceil(float64(size.Height) * h / 100.))
				opts.Left = int(math.Ceil(float64(size.Width) * x / 100.))
				opts.Top = int(math.Ceil(float64(size.Height) * y / 100.))
				opts.AreaWidth += opts.Left
			} else {
				message := fmt.Sprintf(regionError, region)
				return nil, HTTPError{http.StatusBadRequest, message}
			}
		}
		return &opts, nil
	}
	return nil, nil
}
