package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang/groupcache"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	groupcache, _ := ctx.Value(ContextKey("groupcache")).(*groupcache.Group)

	image, modTime, err := openImage(identifier, groupcache)
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

	buffer, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		http.Error(w, "Cannot create profile", http.StatusInternalServerError)
		return
	}

	header := w.Header()

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/ld+json") {
		header.Set("Content-Type", "application/ld+json")
	} else {
		header.Set("Content-Type", "application/json")
	}
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	http.ServeContent(w, r, "info.json", *modTime, bytes.NewReader(buffer))
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
