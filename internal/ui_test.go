package main

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

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
		inEntryIconDim dimensions[int]
		// dimension set by the user
		inputSetHeight float64
		ft             string
		outputDim      [2]float64
	}{
		{"png: iconHeight > setHeight > screenHeight", dimensions{800, 600}, 641, "png", [2]float64{640, 480}},
		{"jpg: iconHeight > setHeight > screenHeight", dimensions{800, 600}, 641, "jpg", [2]float64{640, 480}},
		{"svg: iconHeight > setHeight > screenHeight", dimensions{800, 600}, 641, "svg", [2]float64{640, 480}},
		{"png: screenHeight > iconHeight > setHeight", dimensions{320, 240}, 120, "png", [2]float64{160, 120}},
		{"jpg: screenHeight > iconHeight > setHeight", dimensions{320, 240}, 120, "jpg", [2]float64{160, 120}},
		{"svg: screenHeight > iconHeight > setHeight", dimensions{320, 240}, 120, "svg", [2]float64{160, 120}},
		{"png: setHeight > screenHeight > iconHeight", dimensions{320, 240}, 800, "png", [2]float64{320, 240}},
		{"jpg: setHeight > screenHeight > iconHeight", dimensions{320, 240}, 800, "jpg", [2]float64{320, 240}},
		{"svg: setHeight > screenHeight > iconHeight", dimensions{320, 240}, 800, "svg", [2]float64{640, 480}},
		{"png: screenHeight > setHeight > iconHeight", dimensions{320, 240}, 400, "png", [2]float64{320, 240}},
		{"jpg: screenHeight > setHeight > iconHeight", dimensions{320, 240}, 400, "jpg", [2]float64{320, 240}},
		{"svg: screenHeight > setHeight > iconHeight", dimensions{320, 240}, 400, "svg", [2]float64{533.3333, 400}},
		{"png: iconHeight resized to screenHeight", dimensions{400, 1000}, 800, "png", [2]float64{192, 480}},
		{"jpg: iconHeight resized to screenHeight", dimensions{400, 1000}, 800, "jpg", [2]float64{192, 480}},
		{"svg: iconHeight resized to screenHeight", dimensions{400, 1000}, 800, "svg", [2]float64{192, 480}},
		{"png: iconWidth resized to screenWidth", dimensions{1000, 400}, 800, "png", [2]float64{640, 256}},
		{"jpg: iconWidth resized to screenWidth", dimensions{1000, 400}, 800, "jpg", [2]float64{640, 256}},
		{"svg: iconWidth resized to screenWidth", dimensions{1000, 400}, 800, "svg", [2]float64{640, 256}},
		{"png: iconHeight resized to setHeight", dimensions{400, 200}, 100, "png", [2]float64{200, 100}},
		{"jpg: iconHeight resized to setHeight", dimensions{400, 200}, 100, "jpg", [2]float64{200, 100}},
		{"svg: iconHeight resized to setHeight", dimensions{400, 200}, 100, "svg", [2]float64{200, 100}},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			var e entry
			e.IconHeight.abs = tests[i].inputSetHeight
			icon, err := getMockIcon(tests[i].inEntryIconDim, tests[i].ft)
			defer os.Remove(icon.path)
			if err != nil {
				t.Fatal(err)
			}
			if err := icon.validate(); err != nil {
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
			eImg.img = ebiten.NewImage(int(eImg.dim.width), int(eImg.dim.height))

			x, y := getIconDimensions(&e, eImg)
			if !floatEqual(x, tests[i].outputDim[0]) || !floatEqual(y, tests[i].outputDim[1]) {
				t.Errorf("%s icon: wanted dimensions %f, %f but got %f, %f", tests[i].ft, tests[i].outputDim[0], tests[i].outputDim[1], x, y)
			}
		})
	}
}
