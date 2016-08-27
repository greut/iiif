package profile

import (
	"github.com/thisisaaronland/iiif/image"
)

type Profile struct {
	Context  string   `json:"@profile"`
	Id       string   `json:"@id"`
	Type     string   `json:"@type"` // Optional or iiif:Image
	Protocol string   `json:"protocol"`
	Width    int      `json:"width"`
	Height   int      `json:"height"`
	Profile  []string `json:"profile"`
	//	Sizes    []string `json:"sizes"` // Optional, existing/supported sizes.
	//	Tiles    []string `json:"tiles"` // Optional
}

func NewProfile(host string, im *image.Image) (*Profile, error){

	p := profile.Profile{
		Context:  "http://iiif.io/api/image/2/context.json",
		Id:       fmt.Sprintf("http://%s/%s", host, im.Identifier()),
		Type:     "iiif:Image",
		Protocol: "http://iiif.io/api/image",
		Width:    im.Width(),
		Height:   im.Height(),
		Profile: []string{
			fmt.Sprintf("http://%s/level2.json", host),
		},
	}

	return &p, nil
}