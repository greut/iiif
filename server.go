package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/golang/groupcache"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

var port = flag.String("port", "80", "Define which TCP port to use")
var root = flag.String("root", ".", "Define root directory")
var host = flag.String("host", "0.0.0.0", "Define the hostname")
var db = flag.String("cache", "cache.db", "Define the BoltDB database file")
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

// WithBoltDB sets the context with a cache key.
func WithBoltDB(h http.Handler, db *bolt.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKey("cache"), db)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}

// WithGroupCache sets the cache
func WithGroupCache(h http.Handler, group *groupcache.Group) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKey("groupcache"), group)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}

func makeRouter() http.Handler {
	router := mux.NewRouter()

	router.PathPrefix("/assets").Handler(
		http.StripPrefix("/assets",
			http.FileServer(http.Dir("bower_components"))))
	router.HandleFunc("/{identifier}/info.json", InfoHandler)
	router.HandleFunc("/{identifier}/{region}/{size}/{rotation}/{quality}.{format}", ImageHandler)
	router.HandleFunc("/{identifier}/{viewer}", ViewerHandler)
	router.HandleFunc("/{identifier}", RedirectHandler)
	router.HandleFunc("/", IndexHandler)

	return router
}

func makeHandler() http.Handler {
	router := makeRouter()

	// Caching
	cache, err := bolt.Open(*db, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer cache.Close()

	me := fmt.Sprintf("http://%s/", *host)
	peers := groupcache.NewHTTPPool(me)
	peers.Set(me) // TODO add any other servers here...

	// Group Cache
	var group = groupcache.NewGroup("images", 64<<20, groupcache.GetterFunc(
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

	return WithGroupCache(WithBoltDB(router, cache), group)
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
