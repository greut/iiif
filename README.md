# iiif

This is a fork @greut 's [iiif](https://github.com/greut/iiif) package that moves most of the processing logic in to discrete Go packages and defines source, derivative and graphics details in a JSON config file. There is also an additional caching layer for both source images and derivatives.

_It mostly works but it still a work in progress._

## setup

libvips is required by [bimg](https://github.com/h2non/bimg/). There is a detailed [setup script](ubuntu/setup.sh) available for Ubuntu.

```
$> make bin
$> bin/iiif-server -config config.json
2016/09/01 15:45:07 Serving 127.0.0.1:8080 with pid 12075
```

## config files

There is a [sample config file](config.json.example) included with this repo.

```
{
    "graphics": {
	"source": { "name": "VIPS" }
    },
    "images": {
	"source": { "name": "Disk", "path": "/path/to/images" },
	"cache": { "name": "Memory", "ttl": 300, "limit": 100 }
    },
    "derivatives": {
	"cache": { "name": "Disk", "path": "/path/to/derivatives-cache" }
    }
}
```

_Detailed documentation to follow._

## iiif image API 2.1

The API specifications can be found on [iiif.io](http://iiif.io/api/image/2.1/index.html).

### [Identifier](http://iiif.io/api/image/2.1/#identifier)

* `filename`: the name of the file **(all the images are in one folder)**

### [Region](http://iiif.io/api/image/2.1/index.html#region)

* `full`: the full image
* `square`: a square area in the picture (centered)
* `x,y,w,h`: extract the specified region (as pixels)
* `pct:x,y,w,h`: extract the specified region (as percentages)

### [Size](http://iiif.io/api/image/2.1/index.html#size)

* `full`: the full image **(deprecated)**
* `max`: the full image
* `w,h`: a potentially deformed image of `w x h` **(not supported)**
* `!w,h`: a non-deformed image of maximum `w x h`
* `w,`: a non-deformed image with `w` as the width
* `,h`: a non-deformed image with `h` as the height
* `pct:n`: a non-deformed image scaled by `n` percent

### [Rotate](http://iiif.io/api/image/2.1/index.html#rotation)

* `n` a clockwise rotation of `n` degrees
* `!n` a flip is done before the rotation

__limitations__ bimg only supports rotations that are multiples of 90.

### [Quality](http://iiif.io/api/image/2.1/index.html#quality)

* `color` image in full colour
* `gray` image in grayscale
* `bitonal` image in either black or white pixels **(not supported)**
* `default` image returned in the server default quality

### [Format](http://iiif.io/api/image/2.1/index.html#format)

* `jpg`
* `png`
* `webp`
* `tiff`

__limitations__ : bimg (libvips) doesn't support writing to `jp2`, `gif` or `pdf`.

### [Profile](http://iiif.io/api/image/2.1/#image-information)

It provides all informations but the available `sizes` and `tiles`. The `sizes`
information would be much better linked with a Cache system.

### [Level2 profile](http://iiif.io/api/image/2.1/#profile-description)

It provides meta-informations about the service. **(incomplete)**