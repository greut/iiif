package main

import (
	"flag"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/h2non/bimg.v1"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var port = flag.String("port", "80", "Define which TCP port to use")
var root = flag.String("root", ".", "Define root directory")
var host = flag.String("host", "0.0.0.0", "Define the hostname")

var qualityError = "IIIF 2.1 `quality` and `format` arguments were expected: %#v"
var regionError = "IIIF 2.1 `region` argument is not recognized: %#v"
var sizeError = "IIIF 2.1 `size` argument is not recognized: %#v"
var rotationError = "IIIF 2.1 `rotation` argument is not recognized: %#v"
var rotationMissing = "libvips cannot rotate angle that isn't a multiple of 90: %#v"
var formatError = "IIIf 2.1 `format` argument is not yet recognized: %#v"
var formatMissing = "libvips cannot output this format %#v as of yet"

type Page struct {
	Files []os.FileInfo
}

func main() {
	flag.Parse()

	router := httprouter.New()

	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		log.Println("/")

		files, _ := ioutil.ReadDir(*root)
		p := &Page{Files: files}

		t, _ := template.ParseFiles("templates/index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, p)
	})

	router.GET("/:identifier/:region/:size/:rotation/:quality_format", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		log.Println(fmt.Sprintf("/%v/%v", *root, ps.ByName("identifier")))

		quality_format := ps.ByName("quality_format")
		arr := strings.Split(quality_format, ".")
		if len(arr) != 2 {
			log.Fatalf(qualityError, quality_format)
		}
		quality := arr[0] // default
		format := arr[1] // jpg

		filename := fmt.Sprintf("%v/%v", *root, ps.ByName("identifier"))
		buffer, err := bimg.Read(filename)
		if err != nil {
			log.Printf("Cannot open file %#v: %#v", filename, err.Error())
			http.NotFound(w, r)
			return
		}

		image := bimg.NewImage(buffer)
		size, err := image.Size()
		if err != nil {
			log.Fatal(err)
		}

		// Region
		// ------
		// full
		// square
		// x,y,w,h (in pixels)
		// pct:x,y,w,h (in percents)
		if ps.ByName("region") != "full" {
			region := bimg.Options{
				AreaWidth: size.Width,
				AreaHeight: size.Height,
				Top: 0,
				Left: 0,
			}

			if ps.ByName("region") == "square" {
				if size.Width < size.Height {
					region.Top = (size.Height - size.Width) / 2.
					region.AreaWidth = size.Width
				} else {
					region.Left = (size.Width - size.Height) / 2.
					region.AreaWidth = size.Height
				}
				region.AreaHeight = region.AreaWidth
			} else {
				arr := strings.Split(ps.ByName("region"), ":")
				if len(arr) == 1 {
					sizes := strings.Split(arr[0], ",")
					if len(sizes) != 4 {
						message := fmt.Sprintf(regionError, ps.ByName("region"))
						http.Error(w, message, 400)
						return
					}
					x, _ := strconv.ParseInt(sizes[0], 10, 64)
					y, _ := strconv.ParseInt(sizes[1], 10, 64)
					w, _ := strconv.ParseInt(sizes[2], 10, 64)
					h, _ := strconv.ParseInt(sizes[3], 10, 64)
					region.AreaWidth = int(w)
					region.AreaHeight = int(h)
					region.Left = int(x)
					region.Top = int(y)
				} else if arr[0] == "pct" {
					sizes := strings.Split(arr[1], ",")
					if len(sizes) != 4 {
						message := fmt.Sprintf(regionError, ps.ByName("region"))
						http.Error(w, message, 400)
						return
					}
					x, _ := strconv.ParseFloat(sizes[0], 64)
					y, _ := strconv.ParseFloat(sizes[1], 64)
					w, _ := strconv.ParseFloat(sizes[2], 64)
					h, _ := strconv.ParseFloat(sizes[3], 64)
					region.AreaWidth = int(math.Ceil(float64(size.Width) * w / 100.))
					region.AreaHeight = int(math.Ceil(float64(size.Height) * h / 100.))
					region.Left = int(math.Ceil(float64(size.Width) * x / 100.))
					region.Top = int(math.Ceil(float64(size.Height) * y / 100.))
				} else {
					message := fmt.Sprintf(regionError, ps.ByName("region"))
					http.Error(w, message, 400)
					return
				}
			}

			_, err = image.Process(region)
			if err != nil {
				 log.Fatal(err)
			}
			size = bimg.ImageSize{
				Width: region.AreaWidth,
				Height: region.AreaHeight,
			}
		}

		// Size, Rotation and Quality are made in a single Process call.
		options := bimg.Options{
			Width: size.Width,
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
		if ps.ByName("size") != "max" && ps.ByName("size") != "full" {
			arr := strings.Split(ps.ByName("size"), ":")
			if len(arr) == 1 {
				best := strings.HasPrefix(ps.ByName("size"), "!")
				sizes := strings.Split(strings.Trim(arr[0], "!"), ",")

				if len(sizes) != 2 {
					message := fmt.Sprintf(sizeError, ps.ByName("size"))
					http.Error(w, message, 400)
					return
				}

				wi, err_w := strconv.ParseInt(sizes[0], 10, 64)
				h, err_h := strconv.ParseInt(sizes[1], 10, 64)

				if err_w != nil && err_h != nil {
					message := fmt.Sprintf(sizeError, ps.ByName("size"))
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
				message := fmt.Sprintf(sizeError, ps.ByName("size"))
				http.Error(w, message, 400)
				return
			}
		}

		// Rotation
		// --------
		// n angle clockwise in degrees
		// !n angle clockwise in degrees with a flip (beforehand)
		flip := strings.HasPrefix(ps.ByName("rotation"), "!")
		angle, err := strconv.ParseInt(strings.Trim(ps.ByName("rotation"), "!"), 10, 64)

		if err != nil {
			message := fmt.Sprintf(rotationError, ps.ByName("rotation"))
			http.Error(w, message, 400)
			return
		} else if angle % 90 != 0 {
			message := fmt.Sprintf(rotationMissing, ps.ByName("rotation"))
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
	})

	log.Println(fmt.Sprintf("Server running on %v:%v", *host, *port))
	panic(http.ListenAndServe(fmt.Sprintf("%v:%v", *host, *port), router))
}
