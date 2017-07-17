package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	var port = flag.String("port", "80", "Define which TCP port to use")
	var root = flag.String("root", "public", "Define root directory")
	var host = flag.String("host", "0.0.0.0", "Define the hostname")
	flag.Parse()

	// build router with group cache middleware and root directory.
	handler := setGroupCache(
		WithRootDirectory(makeRouter(), *root),
		fmt.Sprintf("http://%s/", *host), // TODO add any other servers here...
	)

	// Serving
	listen := fmt.Sprintf("%v:%v", *host, *port)

	log.Println(fmt.Sprintf("Server running on %v", listen))
	panic(http.ListenAndServe(listen, handler))
}
