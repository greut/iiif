package iiif

// ImageProfile contains the technical properties about the service.
type ImageProfile struct {
	Context   string   `json:"@context,omitempty"`
	ID        string   `json:"@id,omitempty"`
	Type      string   `json:"@type,omitempty"` // empty or iiif:ImageProfile
	Formats   []string `json:"formats"`
	MaxArea   int      `json:"maxArea,omitempty"`
	MaxHeight int      `json:"maxHeight,omitempty"`
	MaxWidth  int      `json:"maxWidth,omitempty"`
	Qualities []string `json:"qualities"`
	Supports  []string `json:"supports,omitempty"`
}

// Size contains the information for the available sizes
type Size struct {
	Type   string `json:"@type,omitempty"` // empty or iiif:Size
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Tile contains the information to deal with tiles.
type Tile struct {
	Type         string `json:"@type,omitempty"` // empty or iiif:Tile
	ScaleFactors []int  `json:"scaleFactors"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

// Image contains the technical properties about an image.
type Image struct {
	Context  string        `json:"@context"`
	ID       string        `json:"@id"`
	Type     string        `json:"@type,omitempty"` // empty or iiif:Image
	Protocol string        `json:"protocol"`
	Width    int           `json:"width"`
	Height   int           `json:"height"`
	Profile  []interface{} `json:"profile"`
	Sizes    []Size        `json:"sizes,omitempty"`
	Tiles    []Tile        `json:"tiles,omitempty"`
}

// Config stores the IIIF server configuration.
type Config struct {
	Host      string `toml:"host"`
	Port      int    `toml:"port"`
	Templates string `toml:"templates"`
	Images    string `toml:"images"`
}
