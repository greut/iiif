package main

import (
	"encoding/json"
	"expvar"
	"flag"
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gorilla/mux"
	"github.com/thisisaaronland/iiif"
	"github.com/thisisaaronland/iiif/cache"
	"github.com/thisisaaronland/iiif/image"
	"github.com/thisisaaronland/iiif/level"
	"github.com/thisisaaronland/iiif/profile"
	"github.com/thisisaaronland/iiif/source"
	"log"
	"net/http"
	"os"
	"strings"
)

var host = flag.String("host", "localhost", "Define the hostname")
var port = flag.Int("port", 8080, "Define which TCP port to use")
var config = flag.String("config", ".", "config")

var imageSource			   source.Source
var derivativeCache	       iiif.Cache

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

	vars := mux.Vars(r)
	id := vars["identifier"]

	source, err := image.NewImageFromSource(imageSource, id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p, err := profile.NewProfile(*host, source)

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

	vars := mux.Vars(r)
	id := vars["identifier"]

	region := vars["region"]
	size := vars["size"]
	rotation := vars["rotation"]
	quality := vars["quality"]
	format := vars["format"]

	transformation, err := image.NewTransformation(region, size, rotation, quality, format)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	im, err := image.NewImageFromSource(imageSource, id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	derivative, err := im.Transform(transformation)

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

	config, err := iiif.NewConfigFromFile(*config)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
 
	derivativeCache, err = cache.NewCacheFromConfig(config.Derivatives.Cache)

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	imageSource, err = source.NewSourceFromConfig(config.Images)	// fix me - the naming is all weird

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	router := mux.NewRouter()

	router.HandleFunc("/level2.json", ProfileHandler)
	router.HandleFunc("/{identifier}/info.json", InfoHandler)
	router.HandleFunc("/{identifier}/{region}/{size}/{rotation}/{quality}.{format}", ImageHandler)

	expvarHandler := ExpvarHandlerFunc(*host)
	router.HandleFunc("/debug/vars", expvarHandler)

	endpoint := fmt.Sprintf("%s:%d", *host, *port)

	err = gracehttp.Serve(&http.Server{Addr: endpoint, Handler: router})

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	os.Exit(0)
}
