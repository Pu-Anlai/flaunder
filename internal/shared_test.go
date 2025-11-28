package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"slices"
)

// floatEqual compares two floats and returns true if they are within a range
// that would make them considered equal in general usage
func floatEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-4
}

// compareSlices checks whether two slices contain the same elements
// disregarding order
func compareSlices[T comparable](s1, s2 []T) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i := range s1 {
		if !slices.Contains(s2, s1[i]) {
			return false
		}
	}
	return true
}

// getMockApp creates a basic app struct with the necessary values to run tests
func getMockApp() *app {
	a := new(app)
	s := new(settings)
	f := new(font)
	a.entryImgDim.width = 640
	a.entryImgDim.height = 480
	s.TopPadding = measurement{abs: 10}
	s.BottomPadding = measurement{abs: 20}
	s.ImageTitlePadding = measurement{abs: 5}
	a.settings = s
	if err := f.init(0); err != nil {
		panic(err)
	}
	f.height = 15
	a.settings.Font = *f

	return a
}

// getMockSvg returns the byte contents of an svg vector file of a red
// rectangle of size dim
func getMockSvg(dim dimensions) []byte {
	return fmt.Appendf([]byte{},
		`<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
  <rect width="%d" height="%d" fill="red"/>
</svg>`, dim.width, dim.height, dim.width, dim.height)
}

// getMockImage creates an image of a red rectangle of size dim and returns it
func getMockImage(dim dimensions) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, dim.width, dim.height))
	// fill with red
	for x := 0; x < dim.width; x++ {
		for y := 0; y < dim.height; y++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	return img
}

// getMockIcon creates a mock image file and an icon struct, whose path field
// points to the file. REMEMBER to delete the file at icon.path after testing is
// completed
func getMockIcon(dim dimensions, ft string) (*icon, error) {
	// one err variable so we can use it in the switch statement below
	var err error
	tmpFileTempl := fmt.Sprintf("img-file-*.%s", ft)

	tmpFile, err := os.CreateTemp("", tmpFileTempl)
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()

	var img image.Image
	var svg []byte
	if ft == "svg" {
		svg = getMockSvg(dim)
	} else {
		img = getMockImage(dim)
	}

	switch ft {
	case "png":
		err = png.Encode(tmpFile, img)
	case "jpg":
		err = jpeg.Encode(tmpFile, img, nil)
	case "svg":
		_, err = tmpFile.Write(svg)
	default:
		panic(fmt.Sprintf("invalid filetype specified: %s", ft))
	}
	if err != nil {
		return nil, err
	}

	return &icon{
		path: tmpFile.Name(),
	}, nil
}
