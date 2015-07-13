# iiif

A sample and quiet dumb web server to serve pictures following the iiif API.

## iiif image API 2.0

The API specifications can be found on [iiif.io](http://iiif.io/api/image/2.0/index.html).

### Supported

#### [Region](http://iiif.io/api/image/2.0/index.html#region)

* `full`: the full image
* `x,y,w,h`: extract the specified region (as pixels)
* `pct:x,y,w,h`: extract the specified region (as percentages)

#### [Size](http://iiif.io/api/image/2.0/index.html#size)

* `full`: the full image
* `w,h`: a potentially deformed image of `w x h`
* `!w,h`: a non-deformed image of maximum `w x h`
* `w,`: a non-deformed image with `w` as the width
* `,h`: a non-deformed image with `h` as the height
* `pct:n`: a non-deformed image scaled by `n` percent

#### [Rotate](http://iiif.io/api/image/2.0/index.html#rotation)

* `n` a clockwise rotation of `n` degrees
* `!n` a flip is done before the rotation

__limitations__ bimg only supports rotations that are multiples of 90.

#### [Quality](http://iiif.io/api/image/2.0/index.html#quality)

* `color` image in full colour
* `gray` image in grayscale
* `bitonal` image in either black or white pixels
* `default` image returned in the server default quality

__limitations__ bimg only supports color and gray qualities.

#### [Format](http://iiif.io/api/image/2.0/index.html#format)

* `jpg`
* `png`
* `webp`

__limitations__ : bimg (libvips) doesn't support tiff, gif or svg.

### TODO

* Error handling (400 or 500)
* Sendfile
* Caching
