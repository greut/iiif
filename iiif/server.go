package iiif

import (
	"github.com/golang/groupcache"
	"github.com/gorilla/mux"
	"net/http"

	d "github.com/tj/go-debug"
)

var debug = d.Debug("iiif")

// MakeRouter construct the basic router (no middlewares)
func MakeRouter() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/", IndexHandler)
	router.HandleFunc("/demo", DemoHandler)
	router.HandleFunc("/{identifier:.*}/info.json", InfoHandler)
	router.HandleFunc("/{identifier:.*}/{region}/{size}/{rotation}/{quality}.{format}", ImageHandler)
	router.HandleFunc("/{identifier:.*}/{viewer}.html", ViewerHandler)
	router.HandleFunc("/{identifier:.*}", RedirectHandler)

	return router
}

// SetGroupCache set the two caches for input and output pictures
func SetGroupCache(router http.Handler, config *Config, peers ...string) http.Handler {
	// Caching
	pool := groupcache.NewHTTPPool(peers[0])
	pool.Set(peers...)

	var images = groupcache.NewGroup("images", config.Cache.ImagesSize, groupcache.GetterFunc(
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

	var thumbnails = groupcache.NewGroup("thumbnails", config.Cache.ThumbnailsSize, groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			// FIXME ugly bits
			c := ctx.(struct {
				vars   map[string]string
				config *Config
			})
			ci, err := resizeImage(c.config, c.vars, images)
			if err != nil {
				return err
			}

			binTime, _ := ci.ModTime.MarshalBinary()

			debug("Caching %s (%v)", key, ci.ModTime)

			dest.SetProto(&CacheableImage{
				binTime,
				ci.Buffer,
			})
			return nil
		},
	))

	return WithGroupCaches(router, map[string]*groupcache.Group{
		"images":     images,
		"thumbnails": thumbnails,
	})
}
