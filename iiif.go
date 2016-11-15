package main

// IiifImageProfile contains the technical properties about the service.
type IiifImageProfile struct {
	Context   string   `json:"@context,omitempty"`
	ID        string   `json:"@id,omitempty"`
	Type      string   `json:"@type,omitempty"` // empty or iiif:ImageProfile
	Formats   []string `json:"formats"`
	maxArea   int      `json:"maxArea,omitempty"`
	maxHeight int      `json:"maxHeight,omitempty"`
	maxWidth  int      `json:"maxWidth,omitempty"`
	Qualities []string `json:"qualities"`
	Supports  []string `json:"supports,omitempty"`
}

// IiifSize contains the information for the available sizes
type IiifSize struct {
	Type   string `json:"@type,omitempty"` // empty or iiif:Size
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// IiifTile contains the information to deal with tiles.
type IiifTile struct {
	Type         string `json:"@type,omitempty"` // empty or iiif:Tile
	ScaleFactors []int  `json:"scaleFactors"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

// IiifImage contains the technical properties about an image.
type IiifImage struct {
	Context  string        `json:"@context"`
	ID       string        `json:"@id"`
	Type     string        `json:"@type,omitempty"` // empty or iiif:Image
	Protocol string        `json:"protocol"`
	Width    int           `json:"width"`
	Height   int           `json:"height"`
	Profile  []interface{} `json:"profile"`
	Sizes    []IiifSize    `json:"sizes,omitempty"`
	Tiles    []IiifTile    `json:"tiles,omitempty"`
}
