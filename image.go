package main

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/h2non/filetype"
	svg "github.com/h2non/go-is-svg"
)

type icon struct {
	path  string
	image *image.Image
}

// validate returns nil if i points to a supported image file, otherwise it
// returns an appropriate error
func (i *icon) validate() error {
	// not providing an image is allowed:
	if i.path == "" {
		return nil
	}

	buf, err := os.ReadFile(i.path)
	if err != nil {
		return &fileAccessError{path: i.path}
	}

	if !filetype.IsImage(buf) || svg.IsSVG(buf) {
		return &fileAccessError{path: i.path, fileType: "image"}
	}
	return nil
}

func (i *icon) setBaseField(v string) {
	i.path = v
}
