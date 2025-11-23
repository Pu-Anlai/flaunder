package main

import (
	"testing"
)

// getMockApp creates a basic app struct with the necessary values to run tests
func getMockApp() *app {
	a := new(app)
	s := new(settings)
	s.TopPadding = measurement{abs: 10}
	s.BottomPadding = measurement{abs: 20}
	s.ImageTitlePadding = measurement{abs: 5}
	a.settings = s
	a.settings.Font = font{height: 15}
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
