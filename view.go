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

// error messages
var openError = "libvips cannot open this file: %#v"

// encodedURL contains a file URL and its base64 encoded version.
type encodedURL struct {
	URL     *url.URL
	Encoded string
}

type titledURL struct {
	URL   string
	Title string
}

// urlList contains a list of IndexURL.
type urlList []*encodedURL

// IndexData contains the data for the index page.
type demoData struct {
	Files []os.FileInfo
	URLs  urlList
}

// template directory
var templates = "templates"

// template functions
var fns = template.FuncMap{
	"plus1": func(x int) int {
		return x + 1
	},
}

// IndexHandler shows the homepage.
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	p := struct {
		ImagesURL []titledURL
		Viewers   []titledURL
	}{
		ImagesURL: []titledURL{
			{
				"https://www.nasa.gov/sites/default/files/thumbnails/image/blacksea_amo_2017149_lrg.jpg",
				"NASA view of the Black Sea",
			},
			{
				"http://futurist.se/gldt/wp-content/uploads/12.10/gldt1210.png",
				"Linux distributions as of 2010",
			},
			{
				"http://www.acprail.com/images/stories/maps/Swiss_map.jpg",
				"Swiss trains map",
			},
			{
				"http://t0.nl/northern_cardinal_bird_fail.jpg",
				"Northern Cardinal male bird",
			},
		},
		Viewers: []titledURL{
			{"openseadragon.html", "OpenSeadragon"},
			{"leaflet.html", "Leaflet-IIIF"},
			{"iiifviewer.html", "IiifViewer"},
			{"info.json", "JSON-LD profile"},
		},
	}

	t := template.Must(template.New("index.html").Funcs(fns).ParseFiles("templates/index.html"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, p)
}

// DemoHandler responds with the demo page.
func DemoHandler(w http.ResponseWriter, r *http.Request) {
	root := r.Context().Value(ContextKey("root")).(string)
	files, _ := ioutil.ReadDir(root)

	yoan, _ := url.Parse("http://dosimple.ch/yoan.png")

	p := demoData{
		Files: files,
		URLs: urlList{
			{
				yoan,
				base64.StdEncoding.EncodeToString([]byte(yoan.String())),
			},
		},
	}

	t := template.Must(template.ParseFiles("templates/demo.html"))
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

	scheme := "https"

	if r.TLS == nil {
		scheme = "http"
	}
	if r.Header.Get("X-Forwarded-Proto") != "" {
		scheme = r.Header.Get("X-Forwarded-Proto")
	}

	host := r.Host
	if r.Header.Get("X-Forwarded-Host") != "" {
		host = r.Header.Get("X-Forwarded-Host")
	}

	http.Redirect(w, r, fmt.Sprintf("%s://%s/%s/info.json", scheme, host, identifier), 303)
}

// InfoHandler responds to the image technical properties.
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identifier := vars["identifier"]
	identifier, err := url.QueryUnescape(identifier)
	if err != nil {
		log.Printf("Filename is frob %#v", identifier)
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()
	root, _ := ctx.Value(ContextKey("root")).(string)
	groupcache, _ := ctx.Value(ContextKey("groupcache")).(*groupcache.Group)

	identifier = strings.Replace(identifier, "../", "", -1)

	image, modTime, err := openImage(identifier, root, groupcache)
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

	scheme := r.URL.Scheme

	if r.TLS == nil {
		scheme = "http"
	}
	if r.Header.Get("X-Forwarded-Proto") != "" {
		scheme = r.Header.Get("X-Forwarded-Proto")
	}

	host := r.Host
	if r.Header.Get("X-Forwarded-Host") != "" {
		host = r.Header.Get("X-Forwarded-Host")
	}

	p := IiifImage{
		Context:  "http://iiif.io/api/image/2/context.json",
		ID:       fmt.Sprintf("%s://%s/%s", scheme, host, identifier),
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
					"regionSmart", // not part of IIIF
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
	viewer := vars["viewer"] + ".html"
	identifier := vars["identifier"]

	identifier, err := url.QueryUnescape(identifier)
	if err != nil {
		log.Printf("Filename is frob %#v", identifier)
		http.NotFound(w, r)
		return
	}
	identifier = strings.Replace(identifier, "../", "", -1)

	p := &struct{ Image string }{Image: identifier}

	tpl := filepath.Join(templates, "viewer", viewer)
	t, err := template.ParseFiles(tpl)
	if err != nil {
		debug("Template not found. %s. %#v", viewer, err.Error())
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, p)
}
