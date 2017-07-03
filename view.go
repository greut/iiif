package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
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
		URLs: indexURLList{
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

	ctx := r.Context()
	cache, ok := ctx.Value(ContextKey("cache")).(*bolt.DB)

	key := []byte(r.URL.String())
	bucket := []byte("info")

	var buffer []byte
	var err error

	if ok {
		err = cache.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(bucket)
			if b == nil {
				return fmt.Errorf("bucket %q not found", bucket)
			}

			buffer = b.Get(key)
			if buffer == nil {
				return fmt.Errorf("key is empty %q:%q", bucket, key)
			}
			return nil
		})
	}

	if !ok || err != nil {
		image, _, err := openImage(identifier, cache, "")
		if err != nil {
			http.NotFound(w, r)
			return
		}

		size, err := image.Size()
		if err != nil {
			message := fmt.Sprintf(openError, identifier)
			http.Error(w, message, http.StatusNotImplemented)
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
					Formats:   []string{"jpg", "png", "tif", "webp"},
					Qualities: []string{"gray", "default"},
					Supports: []string{
						//"baseUriRedirect",
						//"canonicalLinkHeader",
						"cors",
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

		buffer, err = json.MarshalIndent(p, "", "  ")
		if err != nil {
			log.Fatal("Cannot create profile")
		}

		// Store into cache
		if ok {
			err = cache.Update(func(tx *bolt.Tx) error {
				b, err := tx.CreateBucketIfNotExists(bucket)
				if err != nil {
					return err
				}

				err = b.Put(key, buffer)
				return err
			})

			if err != nil {
				log.Printf("Cannot store %q:%q", key, bucket)
			} else {
				log.Printf("Stored %q:%q", bucket, key)
			}
		}
	}

	header := w.Header()

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/ld+json") {
		header.Set("Content-Type", "application/ld+json")
	} else {
		header.Set("Content-Type", "application/json")
	}
	header.Set("Content-Length", strconv.Itoa(len(buffer)))
	header.Set("Access-Control-Allow-Origin", "*")
	w.Write(buffer)
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

	quality := vars["quality"]
	format := vars["format"]
	identifier := vars["identifier"]
	lastModifiedSince := r.Header.Get("If-Modified-Since")
	rangeContent := r.Header.Get("Range")

	// Content-Type
	contentType := ""
	bimgType := bimg.JPEG
	if format == "jpg" || format == "jpeg" {
		bimgType = bimg.JPEG
		contentType = "image/jpg"
	} else if format == "png" {
		bimgType = bimg.PNG
		contentType = "image/png"
	} else if format == "webp" {
		bimgType = bimg.WEBP
		contentType = "image/webp"
	} else if format == "tif" || format == "tiff" {
		bimgType = bimg.TIFF
		contentType = "image/tiff"
	} else if format == "gif" || format == "pdf" || format == "jp2" {
		message := fmt.Sprintf(formatMissing, format)
		http.Error(w, message, http.StatusNotImplemented)
		return
	}

	if contentType == "" {
		message := fmt.Sprintf(formatError, format)
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	cache, ok := r.Context().Value(ContextKey("cache")).(*bolt.DB)

	key := []byte(r.URL.String())
	bucket := []byte("image")
	bucket2 := []byte("last-modified")

	var buffer []byte
	var lastModified string
	var err error

	if ok {
		err = cache.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(bucket)
			if b == nil {
				return fmt.Errorf("bucket %q not found", bucket)
			}

			buffer = b.Get(key)
			if buffer == nil {
				return fmt.Errorf("cached value is nil %q:%q", bucket, key)
			}

			b = tx.Bucket(bucket2)
			if b == nil {
				return fmt.Errorf("bucket %q not found", bucket2)
			}

			lastModified = string(b.Get(key))
			// if lastModified is empty, we don't care much.
			return nil
		})
	}

	if !ok || err != nil {
		image, stat, err := openImage(identifier, cache, lastModifiedSince)
		if err != nil {
			if err.Error() != "304" {
				http.NotFound(w, r)
			} else {
				code := http.StatusNotModified
				http.Redirect(w, r, http.StatusText(code), code)
			}
			return
		}

		size, err := image.Size()
		if err != nil {
			message := fmt.Sprintf(openError, err.Error())
			http.Error(w, message, http.StatusNotImplemented)
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
						http.Error(w, message, http.StatusBadRequest)
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
						http.Error(w, message, http.StatusBadRequest)
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
					http.Error(w, message, http.StatusBadRequest)
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
					http.Error(w, message, http.StatusBadRequest)
					return
				}

				wi, errW := strconv.ParseInt(sizes[0], 10, 64)
				h, errH := strconv.ParseInt(sizes[1], 10, 64)

				if errW != nil && errH != nil {
					message := fmt.Sprintf(sizeError, s)
					http.Error(w, message, http.StatusBadRequest)
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
				http.Error(w, message, http.StatusBadRequest)
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
			http.Error(w, message, http.StatusBadRequest)
			return
		} else if angle%90 != 0 {
			message := fmt.Sprintf(rotationMissing, rotation)
			http.Error(w, message, http.StatusNotImplemented)
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
			http.Error(w, message, http.StatusBadRequest)
			return
		}

		_, err = image.Process(options)
		if err != nil {
			message := fmt.Sprintf("bimg couldn't process the image: %#v", err.Error())
			http.Error(w, message, http.StatusInternalServerError)
			return
		}

		buffer = image.Image()
		if stat != nil {
			lastModified = stat.ModTime().Format(time.UnixDate)
		}

		// Store into cache
		if ok {
			err = cache.Update(func(tx *bolt.Tx) error {
				b, err := tx.CreateBucketIfNotExists(bucket)
				if err != nil {
					return err
				}

				err = b.Put(key, buffer)
				if err != nil {
					return err
				}

				if lastModified != "" {
					b, err = tx.CreateBucketIfNotExists(bucket2)
					if err != nil {
						return err
					}

					err = b.Put(key, []byte(lastModified))
				}
				return err
			})

			if err != nil {
				log.Printf("Cannot store %q:%q", bucket, key)
			} else {
				log.Printf("Stored %q:%q", bucket, key)
			}
		}
	}

	h := w.Header()
	h.Set("Accept-Ranges", "bytes")
	h.Set("Content-Type", contentType)
	if lastModified != "" {
		h.Set("Last-Modified", lastModified)
	}
	if strings.HasPrefix(rangeContent, "bytes") {
		ranges := strings.Split(rangeContent[6:], "-")

		from, _ := strconv.Atoi(ranges[0])
		to, err := strconv.Atoi(ranges[1])
		if err != nil {
			to = len(buffer) - 1
		}

		// Negative range, e.g. -1024
		if ranges[0] == "" {
			from = len(buffer) - to
			to = len(buffer) - 1
		}

		// Not satisfiable ranges
		if to > len(buffer) || from < 0 {
			h.Set("Content-Range", fmt.Sprintf("bytes */%d", len(buffer)))

			code := http.StatusRequestedRangeNotSatisfiable
			http.Error(w, http.StatusText(code), code)
			return
		}

		h.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", from, to, len(buffer)))
		h.Set("Content-Length", strconv.Itoa(to-from+1))
		buffer = buffer[from : to+1]
		w.WriteHeader(http.StatusPartialContent)
	} else {
		h.Set("Content-Length", strconv.Itoa(len(buffer)))
	}
	_, err = w.Write(buffer)
	if err != nil {
		message := fmt.Sprintf("bimg counldn't write the image: %#v", err.Error())
		http.Error(w, message, http.StatusInternalServerError)
	}
}

func openImage(identifier string, cache *bolt.DB, lastModifiedSince string) (*bimg.Image, os.FileInfo, error) {
	bucket := []byte("remote")
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

		// read from cache
		err = cache.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(bucket)
			if b == nil {
				return fmt.Errorf("bucket %q not found", bucket)
			}
			buffer = b.Get(url)
			if buffer == nil {
				return fmt.Errorf("value is empty %q:%q", bucket, url)
			}
			return nil
		})
		// cache is empty
		if err != nil {
			resp, err := http.Get(string(url))
			if err != nil {
				log.Printf("Download error: %q : %#v.", url, err)
				return nil, nil, err
			}
			defer resp.Body.Close()
			// XXX deal with last-modified-since...
			buf := bytes.NewBuffer(make([]byte, 0, resp.ContentLength))
			_, err = buf.ReadFrom(resp.Body)
			buffer = buf.Bytes()

			// store into cache
			err = cache.Update(func(tx *bolt.Tx) error {
				b, err := tx.CreateBucketIfNotExists(bucket)
				if err != nil {
					return err
				}

				return b.Put(url, buffer)
			})
			if err != nil {
				log.Printf("Cannot update the cache: %q", url)
			}
		} else {
			log.Printf("From cache %q", url)
		}
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
