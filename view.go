package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
