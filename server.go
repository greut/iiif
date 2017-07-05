package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/golang/groupcache"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

var port = flag.String("port", "80", "Define which TCP port to use")
var root = flag.String("root", "public", "Define root directory")
var host = flag.String("host", "0.0.0.0", "Define the hostname")
var templates = "templates"

var openError = "libvips cannot open this file: %#v"
var qualityError = "IIIF 2.1 `quality` and `format` arguments were expected: %#v"
var regionError = "IIIF 2.1 `region` argument is not recognized: %#v"
var sizeError = "IIIF 2.1 `size` argument is not recognized: %#v"
var rotationError = "IIIF 2.1 `rotation` argument is not recognized: %#v"
var rotationMissing = "libvips cannot rotate angle that isn't a multiple of 90: %#v"
var formatError = "IIIF 2.1 `format` argument is not yet recognized: %#v"
var formatMissing = "libvips cannot output this format %#v as of yet"
var multipartRangesNotSupported = "multipart ranges are not supported as of yet."

// ContextKey is the cache key to use.
type ContextKey string

// WithGroupCaches sets the various caches.
func WithGroupCaches(h http.Handler, groups map[string]*groupcache.Group) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		for k, v := range groups {
			ctx = context.WithValue(ctx, ContextKey(k), v)
		}
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}

func makeRouter() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/", IndexHandler)
	router.HandleFunc("/demo", DemoHandler)
	router.HandleFunc("/{identifier}/info.json", InfoHandler)
	router.HandleFunc("/{identifier}/{region}/{size}/{rotation}/{quality}.{format}", ImageHandler)
	router.HandleFunc("/{identifier}/{viewer}", ViewerHandler)
	router.HandleFunc("/{identifier}", RedirectHandler)

	return router
}

func makeHandler() http.Handler {
	router := makeRouter()

	// Caching
	me := fmt.Sprintf("http://%s/", *host)
	peers := groupcache.NewHTTPPool(me)
	peers.Set(me) // TODO add any other servers here...

	var images = groupcache.NewGroup("images", 128<<20, groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			url := key
			data, err := downloadImage(url)
			if err != nil {
				return err
			}
			dest.SetBytes(data)
			return nil
		},
	))

	var thumbnails = groupcache.NewGroup("thumbnails", 512<<20, groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			vars := ctx.(map[string]string)
			data, modTime, err := resizeImage(vars, images)
			if err != nil {
				return err
			}

			binTime, _ := modTime.MarshalBinary()
			dest.SetProto(&ImageWithModTime{binTime, data})
			return nil
		},
	))

	return WithGroupCaches(router, map[string]*groupcache.Group{
		"images":     images,
		"thumbnails": thumbnails,
	})
}

func main() {
	flag.Parse()

	// Routing
	handler := makeHandler()

	// Serving
	listen := fmt.Sprintf("%v:%v", *host, *port)

	log.Println(fmt.Sprintf("Server running on %v", listen))
	panic(http.ListenAndServe(listen, handler))
}
