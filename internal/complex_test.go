package main

import (
	"os"
	"sync"
	"testing"
)

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
		dim  dimensions[int]
		ft   string
	}{
		{"jpg icon 200x400", dimensions[int]{200, 400}, "jpg"},
		{"jpg icon 1200x800", dimensions[int]{1200, 800}, "jpg"},
		{"png icon 200x400", dimensions[int]{200, 400}, "png"},
		{"png icon 1200x800", dimensions[int]{1200, 800}, "png"},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			mockIcon, err := getMockIcon(tests[i].dim, tests[i].ft)
			defer os.Remove(mockIcon.path)
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

func TestIconCalculateWithAspectRatio(t *testing.T) {
	tests := []struct {
		name    string
		iconDim dimensions[int]
		base    float64
		// result with isHeight a) false and b) true
		want [2]float64
	}{
		{"1024x768 -> 800", dimensions[int]{1024, 768}, 640, [2]float64{480, 853.3333}},
		{"200x200 -> 2", dimensions[int]{200, 200}, 2, [2]float64{2, 2}},
		{"2x400 -> 1", dimensions[int]{2, 400}, 1, [2]float64{200, 0.0050}},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			mockIcon, err := getMockIcon(tests[i].iconDim, "jpg")
			if err != nil {
				t.Fatal(err)
			}
			if err := mockIcon.validate(); err != nil {
				t.Fatal(err)
			}
			if err := mockIcon.init(); err != nil {
				t.Fatal(err)
			}

			height := mockIcon.calculateWithAspectRatio(tests[i].base, false)
			width := mockIcon.calculateWithAspectRatio(tests[i].base, true)

			if !floatEqual(height, tests[i].want[0]) {
				t.Errorf("%+v with base width %f wants height %f but got %f", tests[i].iconDim, tests[i].base, tests[i].want[0], height)
			}

			if !floatEqual(width, tests[i].want[1]) {
				t.Errorf("%+v with base height %f wants width %f but got %f", tests[i].iconDim, tests[i].base, tests[i].want[1], width)
			}
		})
	}
}

func TestMeasurement(t *testing.T) {
	var rel1 float64 = 90
	var rel2 float64 = 1440
	var wg sync.WaitGroup

	tests := []struct {
		name      string
		input     measurement
		absolute  *[2]int
		wantError bool
	}{
		{"52%", measurement{value: "52%", abs: 0}, &[2]int{46, 748}, false},
		{"5%", measurement{value: "5%", abs: 0}, &[2]int{4, 72}, false},
		{"100%", measurement{value: "100%", abs: 0}, &[2]int{90, 1440}, false},
		{"2502px", measurement{value: "2502px", abs: 2502}, &[2]int{2502, 2502}, false},
		{"1px", measurement{value: "1px", abs: 1}, &[2]int{1, 1}, false},
		{"", measurement{value: "", abs: 0}, &[2]int{90, 1440}, false},
		{"20p", measurement{value: "20p", abs: 0}, nil, true},
		{"55", measurement{value: "55", abs: 0}, nil, true},
		{"155%", measurement{value: "155%", abs: 0}, nil, true},
		{"-15%", measurement{value: "-15%", abs: 0}, nil, true},
		{"-15px", measurement{value: "-15%", abs: 0}, nil, true},
		{"0%", measurement{value: "0%", abs: 0}, nil, true},
		{"word", measurement{value: "word", abs: 0}, nil, true},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {

			// first test validate method
			err := tests[i].input.validate()
			if tests[i].wantError && err == nil {
				t.Errorf("measurement %v should not parse correctly but did", tests[i].input)
			} else if !tests[i].wantError && err != nil {
				t.Errorf("measurement %v should parse but returned error %s", tests[i].input, err)
			} else if tests[i].wantError {
				// if wantError is true quit here because the init method is
				// expected to fail and will not be run on the struct in the
				// program
				return
			}

			// if validation was succesful, test init method next
			wg.Add(1)
			tests[i].input.init(rel1, &wg)
			if int(tests[i].input.abs) != tests[i].absolute[0] {
				t.Errorf("measurement %v with relative size %f should produce absolute size %d but instead produced %d",
					tests[i].input, rel1, int(tests[i].absolute[0]), int(tests[i].input.abs))
			}
			wg.Add(1)
			tests[i].input.init(rel2, &wg)
			if int(tests[i].input.abs) != tests[i].absolute[1] {
				t.Errorf("measurement %v with relative size %f should produce absolute size %d but instead produced %d",
					tests[i].input, rel2, int(tests[i].absolute[1]), int(tests[i].input.abs))
			}
		})
	}
}
