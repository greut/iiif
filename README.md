# IIIF

[![Build Status](https://travis-ci.org/greut/iiif.svg?branch=master)](https://travis-ci.org/greut/iiif) [![Go Report Card](https://goreportcard.com/badge/github.com/greut/iiif)](https://goreportcard.com/report/github.com/greut/iiif) [![Coverage Status](https://coveralls.io/repos/github/greut/iiif/badge.svg?branch=master)](https://coveralls.io/github/greut/iiif?branch=master)

A sample and quite dumb web server to serve pictures following the [iiif API](http://iiif.io/).

Image API [Compliance](http://iiif.io/api/image/2.1/compliance/) Level 2 is reached.

## Setup

libvips is required by [bimg](https://github.com/h2non/bimg/).

```
$ go get -u \
         github.com/golang/protobuf/proto \
         github.com/golang/groupcache \
         github.com/golang/lint/golint \
         github.com/gorilla/mux \
         github.com/tj/go-debug \
         gopkg.in/h2non/bimg.v1
$ go build -o bin/iiif
$ bin/iiif --help

$ bin/iiif --host 0.0.0.0 --port 8080 --root images

$ DEBUG=iiif,bimg go test -v
```

## IIIF image API 2.1

The API specifications can be found on [iiif.io](http://iiif.io/api/image/2.1/index.html).

### [Identifier](http://iiif.io/api/image/2.1/#identifier)

- `filepath`: the path of the file
- `base64(url)`: the url of the file **(encoded using base64)**

### [Region](http://iiif.io/api/image/2.1/index.html#region)

- `full`: the full image
- `square`: a square area in the picture (centered)
- `x,y,w,h`: extract the specified region (as pixels)
- `pct:x,y,w,h`: extract the specified region (as percentages)
- `smart`: attempt to select the center of interest **(subject to change as it is not part of IIIF)**

### [Size](http://iiif.io/api/image/2.1/index.html#size)

- `full`: the full image **(deprecated)**
- `max`: the full image
- `w,h`: a potentially deformed image of `w x h`
- `!w,h`: a non-deformed image of maximum `w x h`
- `w,`: a non-deformed image with `w` as the width
- `,h`: a non-deformed image with `h` as the height
- `pct:n`: a non-deformed image scaled by `n` percent

### [Rotate](http://iiif.io/api/image/2.1/index.html#rotation)

- `n` a clockwise rotation of `n` degrees
- `!n` a flip is done before the rotation

**limitations** bimg only supports rotations that are multiples of 90.

### [Quality](http://iiif.io/api/image/2.1/index.html#quality)

- `color` image in full colour
- `gray` image in grayscale
- `bitonal` image in either black or white pixels **(not supported)**
- `default` image returned in the server default quality

### [Format](http://iiif.io/api/image/2.1/index.html#format)

- `jpg`
- `png`
- `webp`
- `tiff`

**limitations** : bimg (libvips) doesn't support writing to `gif`, `jp2` or `pdf`.

### [Profile](http://iiif.io/api/image/2.1/#image-information)

It provides all informations but the available `sizes` and `tiles`. The `sizes` information would be much better linked with a Cache system.

### [Level2 profile](http://iiif.io/api/image/2.1/#profile-description)

It provides meta-informations about the service. **(incomplete)**

## Viewers

Some viewers are supporting the iiif API out of the box. The following are included.

- [OpenSeadragon](http://openseadragon.github.io/)
- [Leaflet-IIIF](https://github.com/mejackreed/Leaflet-IIIF)
- [IiifViewer](https://github.com/klokantech/iiifviewer)

## TODO

- Propose maxWidth, maxHeight and maxArea to prevent DOS on the server.
- Sendfile

## Friendly projects

- [thisisaaronland/go-iiif](https://github.com/thisisaaronland/go-iiif)
- [h2non/imaginary](https://github.com/h2non/imaginary)

## Protobuf

```console
$ PATH=$PATH:`pwd`/bin protoc --go_out=. image.proto
```
