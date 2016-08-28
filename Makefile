CWD=$(shell pwd)
GOPATH := $(CWD)

prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep rmdeps
	if test -d src/github.com/thisisaaronland/iiif; then rm -rf src/github.com/thisisaaronland/iiif; fi
	mkdir -p src/github.com/thisisaaronland/iiif
	cp iiif.go src/github.com/thisisaaronland/iiif/
	cp -r cache src/github.com/thisisaaronland/iiif/
	cp -r image src/github.com/thisisaaronland/iiif/
	cp -r level src/github.com/thisisaaronland/iiif/
	cp -r profile src/github.com/thisisaaronland/iiif/
	cp -r source src/github.com/thisisaaronland/iiif/
	cp -r vendor/src/* src/

rmdeps:
	if test -d src; then rm -rf src; fi 

build:	fmt bin

deps:	rmdeps
	@GOPATH=$(GOPATH) go get -u "github.com/facebookgo/grace/gracehttp"
	@GOPATH=$(GOPATH) go get -u "github.com/gorilla/mux"	
	@GOPATH=$(GOPATH) go get -u "github.com/allegro/bigcache"
	@GOPATH=$(GOPATH) go get -u "gopkg.in/h2non/bimg.v1"

vendor: deps
	if test ! -d vendor; then mkdir vendor; fi
	if test -d vendor/src; then rm -rf vendor/src; fi
	cp -r src vendor/src
	find vendor -name '.git' -print -type d -exec rm -rf {} +
	rm -rf src

fmt:
	go fmt *.go
	go fmt cache/*.go
	go fmt image/*.go
	go fmt level/*.go
	go fmt profile/*.go
	go fmt source/*.go

bin: 	self
	@GOPATH=$(GOPATH) go build -o bin/iiif-server cmd/iiif-server.go
