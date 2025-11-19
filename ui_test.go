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
