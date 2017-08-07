GOPATH := $(shell pwd)

.PHONY:
all: bin/iiif

rmdeps:
	test -d src && rm -rf src || true

deps: rmdeps
	mkdir -p src/github.com/greut/iiif
	ln -s ../../../../iiif src/github.com/greut/iiif
	go get -u \
         github.com/BurntSushi/toml \
         github.com/golang/protobuf/proto \
         github.com/golang/groupcache \
         github.com/gorilla/mux \
         github.com/tj/go-debug \
         gopkg.in/h2non/bimg.v1

bin/iiif: iiif/*.go
	go build -o bin/iiif cmd/iiif.go

.PHONY:
test:
	cd iiif; go test -v

.PHONY:
