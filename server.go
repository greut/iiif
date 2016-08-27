package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/thisisaaronland/iiif/image"
	"github.com/thisisaaronland/iiif/level"
	"github.com/thisisaaronland/iiif/profile"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var port = flag.String("port", "80", "Define which TCP port to use")
var root = flag.String("root", ".", "Define root directory")
var cache = flag.String("cache", ".", "Define cache directory")
var host = flag.String("host", "0.0.0.0", "Define the hostname")

var qualityError = "IIIF 2.1 `quality` and `format` arguments were expected: %#v"
var regionError = "IIIF 2.1 `region` argument is not recognized: %#v"
var sizeError = "IIIF 2.1 `size` argument is not recognized: %#v"
var rotationError = "IIIF 2.1 `rotation` argument is not recognized: %#v"
var rotationMissing = "libvips cannot rotate angle that isn't a multiple of 90: %#v"
var formatError = "IIIf 2.1 `format` argument is not yet recognized: %#v"
var formatMissing = "libvips cannot output this format %#v as of yet"

func ProfileHandler(w http.ResponseWriter, r *http.Request) {

     	l, err := level.NewLevel2(r.Host)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(l)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(b)
}

func InfoHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	id := vars["identifier"]
	filename := fmt.Sprintf("%v/%v", *root, id)

	im, err := image.NewImage(*root, id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p, err := profile.NewProfile(host, im)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(p)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(b)
}

func ImageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	rel_path := r.URL.Path
	cache_path := path.Join(*cache, rel_path)

	_, err := os.Stat(cache_path)

	if !os.IsNotExist(err) {

		fmt.Println("read from cache", cache_path)
		body, err := ioutil.ReadFile(cache_path)

		if err != nil {
			panic(err)
		}

		w.Write(body)
		return
	}

	quality := vars["quality"]
	format := vars["format"]

	id := vars["identifier"]

	im, err := image.NewImage(*root, id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	width, height, err := im.Size()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	width := im.Width()
	height := im.Height()
	

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
			} else {
				message := fmt.Sprintf(regionError, region)
				http.Error(w, message, 400)
				return
			}
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

			wi, err_w := strconv.ParseInt(sizes[0], 10, 64)
			h, err_h := strconv.ParseInt(sizes[1], 10, 64)

			if err_w != nil && err_h != nil {
				message := fmt.Sprintf(sizeError, s)
				http.Error(w, message, 400)
				return
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
	if quality == "color" || quality == "default" {
		// do nothing.
	} else if quality == "gray" {
		// FIXME: causes segmentation fault (core dumped)
		//options.Interpretation = bimg.InterpretationGREY16
		options.Interpretation = bimg.InterpretationBW
	} else if quality == "bitonal" {
		options.Interpretation = bimg.InterpretationBW
	} else {
		message := fmt.Sprintf(qualityError, quality)
		http.Error(w, message, 400)
		return
	}

	if format == "jpg" || format == "jpeg" {
		options.Type = bimg.JPEG
		w.Header().Set("Content-Type", "image/jpg")
	} else if format == "png" {
		options.Type = bimg.PNG
		w.Header().Set("Content-Type", "image/png")
	} else if format == "webp" {
		options.Type = bimg.WEBP
		w.Header().Set("Content-Type", "image/webp")
	} else if format == "tif" || format == "tiff" {
		options.Type = bimg.TIFF
		w.Header().Set("Content-Type", "image/tiff")
	} else if format == "gif" || format == "pdf" || format == "jp2" {
		message := fmt.Sprintf(formatMissing, format)
		http.Error(w, message, 501)
		return
	} else {
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

	_, err = w.Write(image.Image())

	if err != nil {
		message := fmt.Sprintf("bimg counldn't write the image: %#v", err.Error())
		http.Error(w, message, 500)
		return
	}

	fmt.Println("set cache", cache_path)
	go func(abs_path string, body []byte) {

		root := filepath.Dir(abs_path)

		_, err := os.Stat(root)

		if os.IsNotExist(err) {
			os.MkdirAll(root, 0755)
		}

		fh, err := os.Create(abs_path)

		if err != nil {
			fmt.Println("failed to cache", abs_path, err)
			return
		}

		defer fh.Close()
		fh.Write(body)
		fh.Sync()

		return

	}(cache_path, image.Image())
}

func main() {
	flag.Parse()

	router := mux.NewRouter()

	router.HandleFunc("/", IndexHandler)
	router.HandleFunc("/level2.json", ProfileHandler)
	router.HandleFunc("/{identifier}/info.json", InfoHandler)
	router.HandleFunc("/{identifier}/{region}/{size}/{rotation}/{quality}.{format}", ImageHandler)

	log.Println(fmt.Sprintf("Server running on %v:%v", *host, *port))
	panic(http.ListenAndServe(fmt.Sprintf("%v:%v", *host, *port), router))
}
