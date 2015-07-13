package main

import (
	"flag"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/h2non/bimg.v0"
	"io"
	"log"
	"math"
	"strconv"
	"strings"
	"net/http"
)

var port = flag.String("port", "80", "Define which TCP port to use.")
var root = flag.String("root", ".", "Define root directory.")
var host = flag.String("host", "0.0.0.0", "Define the hostname.")

func main() {
	flag.Parse()

	router := httprouter.New()

	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		log.Println("/")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, "Hello world!\n")
	})

	router.GET("/:identifier/:region/:size/:rotation/:quality_format", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		log.Println(fmt.Sprintf("/%v/%v", *root, ps.ByName("identifier")))

		quality_format := ps.ByName("quality_format")
		arr := strings.Split(quality_format, ".")
		if len(arr) != 2 {
			log.Fatal("Quality and format were expected. Got: " + quality_format)
		}
		quality := arr[0] // default
		//format := arr[1] // jpg

		buffer, err := bimg.Read(fmt.Sprintf("%v/%v", *root, ps.ByName("identifier")))
		if err != nil {
			log.Fatal(err)
		}

		image := bimg.NewImage(buffer)
		size, err := image.Size()
		if err != nil {
			log.Fatal(err)
		}
		/*
		imageType := bimg.DetermineImageType(buffer)
		if err != err {
			log.Fatal(err)
		}
		log.Println(imageType) // 1:JPEG, 2:WEBP, 3:PNG, 4:TIFF, 5:MAGICK
		*/

		// Region
		// ------
		// full
		// x,y,w,h (in pixels)
		// pct:x,y,w,h (in percents)
		if ps.ByName("region") != "full" {
			region := bimg.Options{
				AreaWidth: size.Width,
				AreaHeight: size.Height,
				Top: 0,
				Left: 0,
			}

			arr := strings.Split(ps.ByName("region"), ":")
			if len(arr) == 1 {
				sizes := strings.Split(arr[0], ",")
				x, _ := strconv.ParseInt(sizes[0], 10, 64)
				y, _ := strconv.ParseInt(sizes[1], 10, 64)
				w, _ := strconv.ParseInt(sizes[2], 10, 64)
				h, _ := strconv.ParseInt(sizes[3], 10, 64)
				region.AreaWidth = int(w)
				region.AreaHeight = int(h)
				region.Top = int(x)
				region.Left = int(y)
			} else if arr[0] == "pct" {
				sizes := strings.Split(arr[1], ",")
				x, _ := strconv.ParseFloat(sizes[0], 64)
				y, _ := strconv.ParseFloat(sizes[1], 64)
				w, _ := strconv.ParseFloat(sizes[2], 64)
				h, _ := strconv.ParseFloat(sizes[3], 64)
				region.AreaWidth = int(math.Ceil(float64(size.Width) * w / 100.))
				region.AreaHeight = int(math.Ceil(float64(size.Height) * h / 100.))
				region.Top = int(math.Ceil(float64(size.Width) * x / 100.))
				region.Left = int(math.Ceil(float64(size.Height) * y / 100.))
			} else {
				log.Fatal("Cannot do anything with " + ps.ByName("region"))
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
			Enlarge: true,
		}

		// Size
		// ----
		// full
		// w,h (deform)
		// !w,h (best fit within size)
		// w, (force width)
		// ,h (force height)
		// pct:n (resize)
		if ps.ByName("size") != "full" {
			arr := strings.Split(ps.ByName("size"), ":")
			if len(arr) == 1 {
				force := strings.HasPrefix(ps.ByName("size"), "!")
				sizes := strings.Split(strings.Trim(arr[0], "!"), ",")

				w, err := strconv.ParseInt(sizes[0], 10, 64)
				if err == nil {
					options.Width = int(w)
				}

				h, err := strconv.ParseInt(sizes[1], 10, 64)
				if err == nil {
					options.Height = int(h)
				}

				if force {
					wr := float64(size.Width) / float64(w)
					hr := float64(size.Height) / float64(h)
					if wr > hr {
						options.Height = 0
					} else {
						options.Width = 0
					}
				}
			} else if arr[0] == "pct" {
				pct, _ := strconv.ParseFloat(arr[1], 64)
				options.Width = int(math.Ceil(pct / 100 * float64(size.Width)))
				options.Height = int(math.Ceil(pct / 100 * float64(size.Height)))
			} else {
				log.Fatal("Cannot do anything with " + ps.ByName("region"))
			}
		}

		// Rotation
		// --------
		// n angle clockwise in degrees
		// !n angle clockwise in degrees with a flip (beforehand)
		if ps.ByName("rotation") != "0" {
			flip := strings.HasPrefix(ps.ByName("rotation"), "!")
			angle, _ := strconv.ParseInt(strings.Trim(ps.ByName("rotation"), "!"), 10, 64)

			angle = angle % 360
			if angle % 90 != 0 {
				log.Fatal(fmt.Sprintf("Cannot rotate angle that aren't multiples of 90: %v", angle))
			}

			options.Flip = flip
			options.Rotate = bimg.Angle(angle)
		}

		// Quality
		// -------
		// color
		// gray
		// bitonal (not supported)
		// default
		if quality == "gray" {
			options.Interpretation = bimg.INTERPRETATION_B_W
		}

		_, err = image.Process(options)
		if err != nil {
			log.Fatal(err)
		}

		// For now jpeg
		w.Header().Set("Content-Type", "image/jpg")

		_, err = w.Write(image.Image())
		if err != nil {
			log.Fatal(err)
		}
	})

	log.Println(fmt.Sprintf("Server running on %v:%v", *host, *port))
	panic(http.ListenAndServe(fmt.Sprintf("%v:%v", *host, *port), router))
}
