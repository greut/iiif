package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/thisisaaronland/iiif/image"
	"github.com/thisisaaronland/iiif/level"
	"github.com/thisisaaronland/iiif/profile"
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

func ExpvarHandlerFunc(host string) http.HandlerFunc {

	f := func(w http.ResponseWriter, r *http.Request) {

		remote := strings.Split(r.RemoteAddr, ":")

		if remote[0] != host {

			http.Error(w, "No soup for you!", http.StatusForbidden)
			return
		}

		// This is copied wholesale from
		// https://golang.org/src/expvar/expvar.go

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintf(w, "{\n")

		first := true

		expvar.Do(func(kv expvar.KeyValue) {
			if !first {
				fmt.Fprintf(w, ",\n")
			}

			first = false
			fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
		})

		fmt.Fprintf(w, "\n}\n")
	}

	return http.HandlerFunc(f)
}

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

	query := r.URL.Query()
	id := query.Get("identifier")

	source, err := image.NewSourceImage(id, sourceCache)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p, err := profile.NewProfile(host, source)

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

	rel_path := r.URL.Path

	body, err := derivativeCache.Get(rel_path)

	if err == nil {

		w.Header().Set("Content-Type", "image/jpg") // FIX ME
		w.Write(body)
		return
	}

	query := r.URL.Query()

	region := query.Get("region")
	size := query.Get("size")
	rotation := query.Get("rotation")
	quality := query.Get("quality")
	format := query.Get("format")

	params, err := image.NewTransformation(region, size, rotation, quality, format)

	if !ok {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	source, err := image.NewSourceImage(id, sourceCache)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	derivative, err := source.Transform(params)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go func(k string, v []byte) {
		derivativeCache.Set(k, v)
	}(rel_path, derivative)

	w.Header().Set("Content-Type", "image/jpg") // FIX ME
	w.Write(body)
	return

	/*

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

	*/
}

func main() {

	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/level2.json", ProfileHandler)
	mux.HandleFunc("/{identifier}/info.json", InfoHandler)
	mux.HandleFunc("/{identifier}/{region}/{size}/{rotation}/{quality}.{format}", ImageHandler)

	expvarHandler := ExpvarHandlerFunc(*host)
	mux.HandleFunc("/debug/vars", expvarHandler)

	endpoint := fmt.Sprintf("%s:%d", *host, *port)

	err := gracehttp.Serve(&http.Server{Addr: endpoint, Handler: mux})

	if err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}
