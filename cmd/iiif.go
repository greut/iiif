package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"code.cloudfoundry.org/bytefmt"
	"github.com/BurntSushi/toml"
	"github.com/greut/iiif/iiif"
)

func main() {
	// Configuration
	var configFile = flag.String("config", "config.toml", "Define the configuration file to use.")
	flag.Parse()

	if flag.NArg() > 0 {
		*configFile = flag.Arg(0)
	}

	var config iiif.Config
	log.Println(fmt.Sprintf("Reading configuration from %s", *configFile))
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		fmt.Println(err)
		return
	}

	iS, _ := bytefmt.ToBytes(config.Cache.Images)
	tS, _ := bytefmt.ToBytes(config.Cache.Thumbnails)
	config.Cache.ImagesSize = int64(iS)
	config.Cache.ThumbnailsSize = int64(tS)

	// build router with root directory.
	handler := iiif.WithConfig(iiif.MakeRouter(), &config)
	// add group cache middleware if the cache size is greater than zero.
	if config.Cache.ImagesSize > 0 && config.Cache.ThumbnailsSize > 0 {
		handler = iiif.SetGroupCache(
			handler,
			&config,
			fmt.Sprintf("http://%s/", config.Host), // TODO add any other servers here...
		)
	}

	// Serving
	listen := fmt.Sprintf("%v:%v", config.Host, config.Port)

	log.Println(fmt.Sprintf("Server running on %v", listen))
	panic(http.ListenAndServe(listen, handler))
}
