package main

import (
	"os"

	"github.com/h2non/filetype"
	svg "github.com/h2non/go-is-svg"
)

type image string

// validate returns nil if img points to a supported image file, otherwise
// it returns an appropriate error
func (img *image) validate() error {
	imgStr := string(*img)
	// not providing an image is allowed:
	if imgStr == "" {
		return nil
	}

	buf, err := os.ReadFile(imgStr)
	if err != nil {
		return &fileAccessError{path: imgStr}
	}
	if !filetype.IsImage(buf) || svg.IsSVG(buf) {
		return &fileAccessError{path: imgStr, fileType: "image"}
	}
	return nil
}
