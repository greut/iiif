package main

import (
	"code.cloudfoundry.org/bytefmt"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/greut/iiif/iiif"
	"log"
	"net/http"
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

	// build router with group cache middleware and root directory.
	handler := iiif.SetGroupCache(
		iiif.WithConfig(iiif.MakeRouter(), &config),
		&config,
		fmt.Sprintf("http://%s/", config.Host), // TODO add any other servers here...
	)

	// Serving
	listen := fmt.Sprintf("%v:%v", config.Host, config.Port)

	log.Println(fmt.Sprintf("Server running on %v", listen))
	panic(http.ListenAndServe(listen, handler))
}
