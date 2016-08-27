package image

import (
	"gopkg.in/h2non/bimg.v1"
	"path/filepath"
)

type Image interface {
     Identifier() string
     Height() int
     Width() int
}

type Image struct {
     source string
     id string
     bimg bimg.Image
}

function NewImage (source string, id string) (*Image, error) {

	 filename := filepath.Join(source, id)

	 buffer, err := bimg.Read(filename)

	 if err != nil {
	    return nil, error
	 }

	 bimg := bimg.NewImage(buffer)

	 image := Image{
	 	source: source,
		id: id,	      
	        bimg: bimg,
	 }

	 return &image, nil
	 size, err := image.Size()
}

func (im *Image) Identifier() string {
     return im.id
}

func (im *Image) Height (int){
	size, _ := im.bimg.Size()
	return size.Height
}

func (im *Image) Width (int){
	size, _ := im.bimg.Size()
	return size.Width
}

func (im *Image) Crop (region string) (error){

     w := im.Width()
     h := im.Height()

		opts := bimg.Options{
			AreaWidth:  w,
			AreaHeight: h,
			Top:        0,
			Left:       0,
		}

		if region == "square" {

			if w < h {
				opts.Top = (h - w) / 2.
				opts.AreaWidth = w
			} else {
				opts.Left = (w - h) / 2.
				opts.AreaWidth = h
			}

			opts.AreaHeight = opts.AreaWidth

		} else {

			arr := strings.Split(region, ":")

			if len(arr) == 1 {

				sizes := strings.Split(arr[0], ",")

				if len(sizes) != 4 {
					message := fmt.Sprintf(regionError, region)
					return errors.New(message)
				}

				x, _ := strconv.ParseInt(sizes[0], 10, 64)
				y, _ := strconv.ParseInt(sizes[1], 10, 64)
				w, _ := strconv.ParseInt(sizes[2], 10, 64)
				h, _ := strconv.ParseInt(sizes[3], 10, 64)

				opts.AreaWidth = int(w)
				opts.AreaHeight = int(h)
				opts.Left = int(x)
				opts.Top = int(y)

			} else if arr[0] == "pct" {

				sizes := strings.Split(arr[1], ",")

				if len(sizes) != 4 {
					message := fmt.Sprintf(regionError, region)
					return errors.New(message)
				}

				x, _ := strconv.ParseFloat(sizes[0], 64)
				y, _ := strconv.ParseFloat(sizes[1], 64)
				w, _ := strconv.ParseFloat(sizes[2], 64)
				h, _ := strconv.ParseFloat(sizes[3], 64)

				opts.AreaWidth = int(math.Ceil(float64(size.Width) * w / 100.))
				opts.AreaHeight = int(math.Ceil(float64(size.Height) * h / 100.))
				opts.Left = int(math.Ceil(float64(size.Width) * x / 100.))
				opts.Top = int(math.Ceil(float64(size.Height) * y / 100.))

			} else {
				message := fmt.Sprintf(regionError, region)
				return errors.New(message)
			}
		}

		_, err = image.Process(opts)

		if err != nil {
			return err
		}

		/*
		size = bimg.ImageSize{
			Width:  opts.AreaWidth,
			Height: opts.AreaHeight,
		}
		*/

		return nil
}