package main

import (
	"github.com/golang/groupcache"
	"github.com/gorilla/mux"
	"net/http"

	d "github.com/tj/go-debug"
)

var debug = d.Debug("iiif")

func makeRouter() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/", IndexHandler)
	router.HandleFunc("/demo", DemoHandler)
	router.HandleFunc("/{identifier:.*}/info.json", InfoHandler)
	router.HandleFunc("/{identifier:.*}/{region}/{size}/{rotation}/{quality}.{format}", ImageHandler)
	router.HandleFunc("/{identifier:.*}/{viewer}.html", ViewerHandler)
	router.HandleFunc("/{identifier:.*}", RedirectHandler)

	return router
}

func setGroupCache(router http.Handler, peers ...string) http.Handler {
	// Caching
	pool := groupcache.NewHTTPPool(peers[0])
	pool.Set(peers...)

	var images = groupcache.NewGroup("images", 128<<20, groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			url := key
			data, err := downloadImage(url)
			if err != nil {
				return err
			}
			debug("Caching %s", key)
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

			debug("Caching %s", key)
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
