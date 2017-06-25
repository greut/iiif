package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/h2non/bimg.v1"
	"html/template"
	"io/ioutil"
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

// IndexURL contains a file URL and its base64 encoded version.
type indexURL struct {
	URL     *url.URL
	Encoded string
}

// IndexURLList contains a list of IndexURL.
type indexURLList []*indexURL

// IndexData contains the data for the index page.
type indexData struct {
	Files []os.FileInfo
	URLs  indexURLList
}

// IndexHandler responds to the service homepage.
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	files, _ := ioutil.ReadDir(*root)

	yoan, _ := url.Parse("http://dosimple.ch/yoan.png")

	p := indexData{
		Files: files,
		Urls: indexURLList{
			{
				yoan,
				base64.StdEncoding.EncodeToString([]byte(yoan.String())),
			},
		},
	}

	t, _ := template.ParseFiles("templates/index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, &p)
}

// RedirectHandler responds to the image technical properties.
func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	identifier := vars["identifier"]
	identifier, err := url.QueryUnescape(identifier)
	if err != nil {
		log.Printf("Filename is frob %#v", identifier)
		http.NotFound(w, r)
		return
	}

	identifier = strings.Replace(identifier, "../", "", -1)

	http.Redirect(w, r, fmt.Sprintf("%s://%s/%s/info.json", r.URL.Scheme, r.Host, identifier), 303)
}

// InfoHandler responds to the image technical properties.
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identifier := vars["identifier"]
	image, _, err := openImage(identifier, "")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	size, err := image.Size()
	if err != nil {
		message := fmt.Sprintf(openError, identifier)
		http.Error(w, message, 501)
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}

	p := IiifImage{
		Context:  "http://iiif.io/api/image/2/context.json",
		ID:       fmt.Sprintf("%s://%s/%s", scheme, r.Host, identifier),
		Type:     "iiif:Image",
		Protocol: "http://iiif.io/api/image",
		Width:    size.Width,
		Height:   size.Height,
		Profile: []interface{}{
			"http://iiif.io/api/image/2/level2.json",
			&IiifImageProfile{
				Context:   "http://iiif.io/api/image/2/context.json",
				Type:      "iiif:ImageProfile",
				Formats:   []string{"jpg", "png", "webp"},
				Qualities: []string{"gray", "default"},
				Supports: []string{
					//"baseUriRedirect",
					//"canonicalLinkHeader",
					//"cors",
					"jsonldMediaType",
					"mirroring",
					//"profileLinkHeader",
					"regionByPct",
					"regionByPx",
					"regionSquare",
					//"rotationArbitrary",
					"rotationBy90s",
					"sizeAboveFull",
					"sizeByConfinedWh",
					"sizeByDistortedWh",
					"sizeByH",
					"sizeByPct",
					"sizeByW",
					"sizeByWh",
				},
			},
		},
	}

	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		log.Fatal("Cannot create profile")
	}
	header := w.Header()

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/ld+json") {
		header.Set("Content-Type", "application/ld+json")
	} else {
		header.Set("Content-Type", "application/json")
	}
	header.Set("access-control-allow-origin", "*")
	w.Write(b)
}

// ViewerHandler responds with the existing templates.
func ViewerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	viewer := vars["viewer"]
	identifier := vars["identifier"]

	identifier, err := url.QueryUnescape(identifier)
	if err != nil {
		log.Printf("Filename is frob %#v", identifier)
		http.NotFound(w, r)
		return
	}
	identifier = strings.Replace(identifier, "../", "", -1)

	filename := filepath.Join(*root, identifier)
	_, err = os.Stat(filename)
	if err != nil {
		log.Printf("Cannot open file %#v: %#v", filename, err.Error())
		http.NotFound(w, r)
		return
	}

	p := &struct{ Image string }{Image: identifier}

	tpl := filepath.Join(templates, viewer)
	t, err := template.ParseFiles(tpl)
	if err != nil {
		log.Printf("Template not found. %#v", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, p)
}

// ImageHandler responds to the IIIF 2.1 Image API.
func ImageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	//log.Printf("vars %#v", vars)

	quality := vars["quality"]
	format := vars["format"]
	identifier := vars["identifier"]
	lastModifiedSince := r.Header.Get("If-Modified-Since")

	image, stat, err := openImage(identifier, lastModifiedSince)
	if err != nil {
		if err.Error() != "304" {
			http.NotFound(w, r)
		} else {
			http.Redirect(w, r, "Not Modified", 304)
		}
		return
	}

	size, err := image.Size()
	if err != nil {
		message := fmt.Sprintf(openError, err.Error())
		http.Error(w, message, 501)
		return
	}

	// Region
	// ------
	// full
	// square
	// x,y,w,h (in pixels)
	// pct:x,y,w,h (in percents)
	region := vars["region"]
	if region != "full" {
		opts := bimg.Options{
			AreaWidth:  size.Width,
			AreaHeight: size.Height,
			Top:        0,
			Left:       0,
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
					http.Error(w, message, 400)
					return
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
					http.Error(w, message, 400)
					return
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
				http.Error(w, message, 400)
				return
			}
		}

		// Hack: libvips does strange things here.
		// * https://github.com/h2non/bimg/issues/60
		// * https://github.com/h2non/bimg/commit/b7eaa00f104a8eab49eedf49d75b11308df95f7a
		if opts.Top <= 0 && opts.Left == 0 {
			opts.Top = -1
		}

		_, err = image.Process(opts)
		if err != nil {
			log.Fatal(err)
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
				http.Error(w, message, 400)
				return
			}

			wi, errW := strconv.ParseInt(sizes[0], 10, 64)
			h, errH := strconv.ParseInt(sizes[1], 10, 64)

			if errW != nil && errH != nil {
				message := fmt.Sprintf(sizeError, s)
				http.Error(w, message, 400)
				return
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
			http.Error(w, message, 400)
			return
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
		http.Error(w, message, 400)
		return
	} else if angle%90 != 0 {
		message := fmt.Sprintf(rotationMissing, rotation)
		http.Error(w, message, 501)
		return
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
		http.Error(w, message, 400)
		return
	}

	contentType := ""
	if format == "jpg" || format == "jpeg" {
		options.Type = bimg.JPEG
		contentType = "image/jpg"
	} else if format == "png" {
		options.Type = bimg.PNG
		contentType = "image/png"
	} else if format == "webp" {
		options.Type = bimg.WEBP
		contentType = "image/webp"
	} else if format == "tif" || format == "tiff" {
		options.Type = bimg.TIFF
		contentType = "image/tiff"
	} else if format == "gif" || format == "pdf" || format == "jp2" {
		message := fmt.Sprintf(formatMissing, format)
		http.Error(w, message, 501)
		return
	}

	if contentType == "" {
		message := fmt.Sprintf(formatError, format)
		http.Error(w, message, 400)
		return
	}

	_, err = image.Process(options)
	if err != nil {
		message := fmt.Sprintf("bimg couldn't process the image: %#v", err.Error())
		http.Error(w, message, 500)
		return
	}

	buf := image.Image()

	h := w.Header()
	h.Set("Content-Type", contentType)
	h.Set("Content-Length", strconv.Itoa(len(buf)))
	if stat != nil {
		h.Set("Last-Modified", stat.ModTime().Format(time.UnixDate))
	}
	_, err = w.Write(buf)
	if err != nil {
		message := fmt.Sprintf("bimg counldn't write the image: %#v", err.Error())
		http.Error(w, message, 500)
	}
}

func openImage(identifier, lastModifiedSince string) (*bimg.Image, os.FileInfo, error) {
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

		resp, err := http.Get(string(url))
		if err != nil {
			log.Printf("Download error: %#v.", err)
			return nil, nil, err
		}
		defer resp.Body.Close()
		// XXX deal with last-modified-since...
		buf := bytes.NewBuffer(make([]byte, 0, resp.ContentLength))
		_, err = buf.ReadFrom(resp.Body)
		buffer = buf.Bytes()
	} else {
		if lastModifiedSince != "" {
			t, _ := time.Parse(time.UnixDate, lastModifiedSince)
			if err == nil {
				if !t.Before(stat.ModTime()) {
					return nil, nil, errors.New("304")
				}
			}
		}

		buffer, err = bimg.Read(filename)
		if err != nil {
			log.Printf("Cannot open file %#v: %#v", filename, err.Error())
			return nil, nil, err
		}
	}

	image := bimg.NewImage(buffer)
	return image, stat, nil
}
