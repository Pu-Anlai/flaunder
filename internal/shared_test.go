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
	s.IconTitlePadding = measurement{abs: 5}
	a.settings = s

	if err := f.validate(); err != nil {
		panic(err)
	}
	if err := f.init(0); err != nil {
		panic(err)
	}

	f.height = 15
	a.settings.Font = *f

	return a
}

// getMockSvg returns the byte contents of an svg vector file of a red
// rectangle of size dim
func getMockSvg(dim dimensions[int]) []byte {
	return fmt.Appendf([]byte{},
		`<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
  <rect width="%d" height="%d" fill="red"/>
</svg>`, dim.width, dim.height, dim.width, dim.height)
}

// getMockImage creates an image of a red rectangle of size dim and returns it
func getMockImage(dim dimensions[int]) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, dim.width, dim.height))
	// fill with red
	for x := 0; x < dim.width; x++ {
		for y := 0; y < dim.height; y++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	return img
}

// getTmpFile creates a temporary file with the extension ext and returns its
// file object. Not being able to create the temp file will cause a panic.
// Remember to close and delete the file.
func getTmpFile(ext string) *os.File {
	// add a dot in front of the extension string if one was passed
	if ext != "" {
		ext = "." + ext
	}
	template := fmt.Sprintf("flunder-tmpfile-*%s", ext)
	tmpFile, err := os.CreateTemp("", template)
	if err != nil {
		panic(err)
	}
	return tmpFile
}

// getMockIcon creates a mock image file and an icon struct, whose path field
// points to the file. REMEMBER to delete the file at icon.path after testing is
// completed
func getMockIcon(dim dimensions[int], ft string) (*icon, error) {
	tmpFile := getTmpFile(ft)
	defer tmpFile.Close()

	var img image.Image
	var svg []byte
	if ft == "svg" {
		svg = getMockSvg(dim)
	} else {
		img = getMockImage(dim)
	}

	var err error // reusable err variable
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
