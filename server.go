package main

import (
	"flag"
	"fmt"
	"github.com/golang/groupcache"
	"github.com/gorilla/mux"
	"log"
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

func main() {
	var port = flag.String("port", "80", "Define which TCP port to use")
	var root = flag.String("root", "public", "Define root directory")
	var host = flag.String("host", "0.0.0.0", "Define the hostname")

	flag.Parse()

	me := fmt.Sprintf("http://%s/", *host)

	// Routing
	router := makeRouter()
	// TODO add any other servers here...
	handler := setGroupCache(router, me)
	// Sets the root directory.
	handler = WithRootDirectory(router, *root)

	// Serving
	listen := fmt.Sprintf("%v:%v", *host, *port)

	log.Println(fmt.Sprintf("Server running on %v", listen))
	panic(http.ListenAndServe(listen, handler))
}
