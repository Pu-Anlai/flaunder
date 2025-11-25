package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

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

func TestValidateDimensions(t *testing.T) {
	a := getMockApp()
	tests := []struct {
		name      string
		input     int
		wantError bool
	}{
		{"screen too small", 40, true},
		{"screen big enough", 200, false},
		{"screen exactly big enough", 50, false},
		{"screen barely too small", 49, true},
		{"screen barely big enough", 51, false},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			a.screenDim.height = tests[i].input
			err := a.validateDimensions()
			if (err != nil) != tests[i].wantError {
				t.Errorf("screen size %d has wantError %t which it did not produce", tests[i].input, tests[i].wantError)
			}

		})
	}
}

func TestGetIconDimensions(t *testing.T) {
	a := getMockApp()
	tests := []struct {
		name string
		// original dimensions of the icon in entry
		inEntryIconDim dimensions
		// dimension set by the user
		inputSetHeight float64
		outputDim      [2]float64
	}{
		{"iconHeight > setHeight > screenHeight", dimensions{800, 600}, 641, [2]float64{640, 480}},
		{"iconHeight > setHeight < screenHeight", dimensions{320, 240}, 120, [2]float64{160, 120}},
		{"iconHeight < setHeight > screenHeight", dimensions{320, 240}, 800, [2]float64{320, 240}},
		{"iconHeight < setHeight < screenHeight", dimensions{320, 240}, 800, [2]float64{320, 240}},
		{"iconHeight resized to screenHeight", dimensions{400, 1000}, 800, [2]float64{192, 480}},
		{"iconWidth resized to screenWidth", dimensions{1000, 400}, 800, [2]float64{640, 256}},
		{"iconHeight resized to setHeight", dimensions{400, 200}, 100, [2]float64{200, 100}},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			var e entry
			e.IconHeight.abs = tests[i].inputSetHeight
			icon, err := getMockIcon(tests[i].inEntryIconDim, "png")
			defer os.Remove(icon.path)
			// TODO: add tests for svg icons
			if err != nil {
				t.Fatal(err)
			}
			if err := icon.init(); err != nil {
				t.Fatal(err)
			}

			// store icon in entry e
			e.Icon = *icon
			// get a new entryImg that uses app's dimensions
			eImg := a.newEntryImg(tests[i].name)
			// create a new ebiten image with app dimensions and store it in
			// eImg.img
			eImg.img = ebiten.NewImage(eImg.dim.width, eImg.dim.height)

			x, y := getIconDimensions(&e, eImg)
			if x != tests[i].outputDim[0] || y != tests[i].outputDim[1] {
				t.Errorf("wanted dimensions %f, %f but got %f, %f", tests[i].outputDim[0], tests[i].outputDim[1], x, y)
			}
		})
	}
}
