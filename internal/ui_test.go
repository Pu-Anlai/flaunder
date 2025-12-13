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
		// remember that the max size of the icon will still leave room for the icon-title padding and the title height
		{"png: iconHeight > setHeight > screenHeight", dimensions[int]{800, 600}, 641, "png", [2]float64{613.3333, 460}},
		{"jpg: iconHeight > setHeight > screenHeight", dimensions[int]{800, 600}, 641, "jpg", [2]float64{613.3333, 460}},
		{"svg: iconHeight > setHeight > screenHeight", dimensions[int]{800, 600}, 641, "svg", [2]float64{613.3333, 460}},
		{"png: screenHeight > iconHeight > setHeight", dimensions[int]{320, 240}, 120, "png", [2]float64{160, 120}},
		{"jpg: screenHeight > iconHeight > setHeight", dimensions[int]{320, 240}, 120, "jpg", [2]float64{160, 120}},
		{"svg: screenHeight > iconHeight > setHeight", dimensions[int]{320, 240}, 120, "svg", [2]float64{160, 120}},
		{"png: setHeight > screenHeight > iconHeight", dimensions[int]{320, 240}, 800, "png", [2]float64{320, 240}},
		{"jpg: setHeight > screenHeight > iconHeight", dimensions[int]{320, 240}, 800, "jpg", [2]float64{320, 240}},
		{"svg: setHeight > screenHeight > iconHeight", dimensions[int]{320, 240}, 800, "svg", [2]float64{613.3333, 460}},
		{"png: screenHeight > setHeight > iconHeight", dimensions[int]{320, 240}, 400, "png", [2]float64{320, 240}},
		{"jpg: screenHeight > setHeight > iconHeight", dimensions[int]{320, 240}, 400, "jpg", [2]float64{320, 240}},
		{"svg: screenHeight > setHeight > iconHeight", dimensions[int]{320, 240}, 400, "svg", [2]float64{533.3333, 400}},
		{"png: iconHeight resized to screenHeight", dimensions[int]{400, 1000}, 800, "png", [2]float64{184, 460}},
		{"jpg: iconHeight resized to screenHeight", dimensions[int]{400, 1000}, 800, "jpg", [2]float64{184, 460}},
		{"svg: iconHeight resized to screenHeight", dimensions[int]{400, 1000}, 800, "svg", [2]float64{184, 460}},
		{"png: iconWidth resized to screenWidth", dimensions[int]{1000, 400}, 800, "png", [2]float64{640, 256}},
		{"jpg: iconWidth resized to screenWidth", dimensions[int]{1000, 400}, 800, "jpg", [2]float64{640, 256}},
		{"svg: iconWidth resized to screenWidth", dimensions[int]{1000, 400}, 800, "svg", [2]float64{640, 256}},
		{"png: iconHeight resized to setHeight", dimensions[int]{400, 200}, 100, "png", [2]float64{200, 100}},
		{"jpg: iconHeight resized to setHeight", dimensions[int]{400, 200}, 100, "jpg", [2]float64{200, 100}},
		{"svg: iconHeight resized to setHeight", dimensions[int]{400, 200}, 100, "svg", [2]float64{200, 100}},
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

			iconDim := getIconDimensions(&e, eImg)
			if !floatEqual(iconDim.width, tests[i].outputDim[0]) || !floatEqual(iconDim.height, tests[i].outputDim[1]) {
				t.Errorf("%s icon: wanted dimensions %+v but got %+v", tests[i].ft, tests[i].outputDim, iconDim)
			}
		})
	}
}

func TestGetIconOriginPoint(t *testing.T) {
	a := getMockApp()
	tests := []struct {
		name         string
		ebitenImgDim dimensions[float64]
		iconDim      dimensions[float64]
		// for simplicity, we're always using the IconTitlePadding value
		// defined in getMockApp()
		want [2]float64
	}{
		{"ebitenImgSize: 640x480, iconSize: 200x200", dimensions[float64]{640, 480}, dimensions[float64]{200, 200}, [2]float64{220, 130}},
		{"ebitenImgSize: 640x480, iconSize: 640x460", dimensions[float64]{640, 480}, dimensions[float64]{640, 460}, [2]float64{0, 0}},
		{"ebitenImgSize: 640x480, iconSize: 300x30", dimensions[float64]{640, 480}, dimensions[float64]{300, 30}, [2]float64{170, 215}},
		{"ebitenImgSize: 640x480, iconSize: 30x300", dimensions[float64]{640, 480}, dimensions[float64]{30, 300}, [2]float64{305, 80}},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {

			a.entryImgDim = tests[i].ebitenImgDim
			eImg := a.newEntryImg(tests[i].name)
			x, y := getIconOriginPoint(eImg, tests[i].iconDim)

			if !floatEqual(x, tests[i].want[0]) || !floatEqual(y, tests[i].want[1]) {
				t.Errorf("want %+v but got %+v", tests[i].want, [2]float64{x, y})
			}
		})
	}
}

func TestGetTitleOriginPoint(t *testing.T) {
	a := getMockApp()
	tests := []struct {
		name         string
		ebitenImgDim dimensions[float64]
		titleDim     dimensions[float64]
		// for simplicity, we're always using the IconTitlePadding value
		// defined in getMockApp()
		want [2]float64
	}{
		{"ebitenImgSize: 640x480, titleSize: 640x20", dimensions[float64]{640, 480}, dimensions[float64]{640, 20}, [2]float64{0, 460}},
		{"ebitenImgSize: 640x480, titleSize: 700x10", dimensions[float64]{640, 480}, dimensions[float64]{700, 10}, [2]float64{-30, 470}},
		{"ebitenImgSize: 640x480, titleSize: 50x200", dimensions[float64]{640, 480}, dimensions[float64]{50, 200}, [2]float64{295, 280}},
		{"ebitenImgSize: 640x480, titleSize: 400x50", dimensions[float64]{640, 480}, dimensions[float64]{400, 50}, [2]float64{120, 430}},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {

			a.entryImgDim = tests[i].ebitenImgDim
			eImg := a.newEntryImg(tests[i].name)
			eImg.titleDim = tests[i].titleDim
			x, y := getTitleOriginPoint(eImg)

			if !floatEqual(x, tests[i].want[0]) || !floatEqual(y, tests[i].want[1]) {
				t.Errorf("want %+v but got %+v", tests[i].want, [2]float64{x, y})
			}
		})
	}
}

// TestVisualConfirm is not a real test but a helper function for visual
// confirmation. It should be run isolated by passing its name to the -run flag
// of the go test utility. To avoid the function being executed during regular
// test runs, it will return immediately unless the environment variable
// FLUNDER_VISUAL_TEST is set to "yes".
func TestVisualConfirm(t *testing.T) {
	if os.Getenv("FLUNDER_VISUAL_TEST") != "yes" {
		return
	}
}
