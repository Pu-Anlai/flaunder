package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

// getMockIcon creates a mock image file and an icon struct, whose path field
// points to the file. REMEMBER to delete the file at icon.path after testing is
// completed
func getMockIcon(dim dimensions, ft string) (*icon, error) {
	// one err variable so we can use it in the switch statement below
	var err error
	tmpFileTempl := fmt.Sprintf("img-file-*.%s", ft)
	img := image.NewRGBA(image.Rect(0, 0, dim.width, dim.height))
	// fill with red
	for x := 0; x < dim.width; x++ {
		for y := 0; y < dim.height; y++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	tmpFile, err := os.CreateTemp("", tmpFileTempl)
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()

	switch ft {
	case "png":
		err = png.Encode(tmpFile, img)
	case "jpg":
		err = jpeg.Encode(tmpFile, img, nil)
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

func TestIconValidate(t *testing.T) {
	// test non existing paths throwing an error, otherwise validation should be
	// covered by TestIconInit
	tmpFile, err := os.CreateTemp("", "throw-away-*")
	if err != nil {
		t.Fatal(err)
	}
	path := tmpFile.Name()

	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	i := icon{path: path}
	if err := i.validate(); err == nil {
		t.Error("icon did not fail to validate despite invalid path")
	}

}

func TestIconInit(t *testing.T) {
	tests := []struct {
		name string
		dim  dimensions
		ft   string
	}{
		{"test name", dimensions{200, 400}, "jpg"},
		{"test name", dimensions{1200, 800}, "jpg"},
		{"test name", dimensions{200, 400}, "png"},
		{"test name", dimensions{1200, 800}, "png"},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			mockIcon, err := getMockIcon(tests[i].dim, tests[i].ft)
			if err != nil {
				t.Fatal(err)
			}

			err = mockIcon.validate()
			if err != nil {
				t.Fatal(err)
			}

			err = mockIcon.init()
			if err != nil {
				t.Fatal(err)
			}

			if mockIcon.dim != tests[i].dim {
				t.Errorf("icon created with dimensions %d x %d did not initialize to image of same size", tests[i].dim.width, tests[i].dim.height)
			}
		})
	}
}

func TestMeasurementValidate(t *testing.T) {
	tests := []struct {
		name      string
		input     measurement
		wantError bool
	}{
		{"52%", measurement{value: "52%", abs: 0}, false},
		{"5%", measurement{value: "5%", abs: 0}, false},
		{"100%", measurement{value: "100%", abs: 0}, false},
		{"2502px", measurement{value: "2502px", abs: 0}, false},
		{"1px", measurement{value: "1px", abs: 0}, false},
		{"", measurement{value: "", abs: 0}, false},
		{"20p", measurement{value: "20p", abs: 0}, true},
		{"55", measurement{value: "55", abs: 0}, true},
		{"155%", measurement{value: "155%", abs: 0}, true},
		{"-15%", measurement{value: "-15%", abs: 0}, true},
		{"0%", measurement{value: "0%", abs: 0}, true},
		{"word", measurement{value: "word", abs: 0}, true},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {

			err := tests[i].input.validate()
			if tests[i].wantError && err == nil {
				t.Errorf("measurement %v should not parse correctly but did", tests[i].input)
			} else if !tests[i].wantError && err != nil {
				t.Errorf("measurement %v should parse but returned error %s", tests[i].input, err)
			}
		})
	}
}
