package image

import (
	"gopkg.in/h2non/bimg.v1"
	"path/filepath"
	"strings"
	"strconv"
)

type SourceImage struct {
     Image
     source string
     id string
     bimg bimg.Image
}

function NewSourceImage (source string, id string) (*SourceImage, error) {

	 filename := filepath.Join(source, id)

	 buffer, err := bimg.Read(filename)

	 if err != nil {
	    return nil, error
	 }

	 bimg := bimg.NewImage(buffer)

	 source := SourceImage{
	 	source: source,
		id: id,	      
	        bimg: bimg,
	 }

	 return &source, nil
}

func (src *SourceImage) Identifier() string {
     return src.id
}

func (src *SourceImage) Height (int){
	size, _ := src.bimg.Size()
	return size.Height
}

func (src *SourceImage) Width (int){
	size, _ := src.bimg.Size()
	return size.Width
}

func (src *SourceImage) Tranform (t *image.Transformation) ([]byte, error){

	if region != "full" {
     w := src.Width()
     h := src.Height()

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

		_, err = src.bimg.Process(opts)

		if err != nil {
			return err
		}

		/*
		size = bimg.ImageSize{
			Width:  opts.AreaWidth,
			Height: opts.AreaHeight,
		}
		*/
	
	// Size, Rotation and Quality are made in a single Process call.
	options := bimg.Options{
		Width:  size.Width,
		Height: size.Height,
	}

	if s != "max" && s != "full" {
		arr := strings.Split(s, ":")
		if len(arr) == 1 {
			best := strings.HasPrefix(s, "!")
			sizes := strings.Split(strings.Trim(arr[0], "!"), ",")

			if len(sizes) != 2 {
				message := fmt.Sprintf(sizeError, s)
				http.Error(w, message, 400)
				return
			}

			wi, err_w := strconv.ParseInt(sizes[0], 10, 64)
			h, err_h := strconv.ParseInt(sizes[1], 10, 64)

			if err_w != nil && err_h != nil {
				message := fmt.Sprintf(sizeError, s)
				http.Error(w, message, 400)
				return
			} else if err_w == nil && err_h == nil {
				options.Width = int(wi)
				options.Height = int(h)
				if best {
					options.Enlarge = true
				} else {
					options.Force = true
				}
			} else if err_h != nil {
				options.Width = int(wi)
				options.Height = 0
			} else {
				options.Width = 0
				options.Height = int(h)
			}
		} else if arr[0] == "pct" {
			pct, _ := strconv.ParseFloat(arr[1], 64)
			options.Width = int(math.Ceil(pct / 100 * float64(size.Width)))
			options.Height = int(math.Ceil(pct / 100 * float64(size.Height)))
		} else {
			message := fmt.Sprintf(sizeError, s)
			http.Error(w, message, 400)
			return
		}
	}

	flip := strings.HasPrefix(rotation, "!")
	angle, err := strconv.ParseInt(strings.Trim(rotation, "!"), 10, 64)

	if err != nil {
		message := fmt.Sprintf(rotationError, rotation)
		http.Error(w, message, 400)
		return
	} else if angle%90 != 0 {
		message := fmt.Sprintf(rotationMissing, rotation)
		http.Error(w, message, 501)
		return
	}

	options.Flip = flip
	options.Rotate = bimg.Angle(angle % 360)

	if quality == "color" || quality == "default" {
		// do nothing.
	} else if quality == "gray" {
		// FIXME: causes segmentation fault (core dumped)
		//options.Interpretation = bimg.InterpretationGREY16
		options.Interpretation = bimg.InterpretationBW
	} else if quality == "bitonal" {
		options.Interpretation = bimg.InterpretationBW
	} else {
		message := fmt.Sprintf(qualityError, quality)
		http.Error(w, message, 400)
		return
	}
}

	bytes, err = image.Process(options)

	if err != nil {
		message := fmt.Sprintf("bimg couldn't process the image: %#v", err.Error())
		http.Error(w, message, 500)
		return
	}

	return bytes, nil
}