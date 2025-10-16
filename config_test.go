package main

import (
	"os"
	"slices"
	"testing"

	"gopkg.in/ini.v1"
)

var validIniFile []byte = []byte(
	`[Global]
Background = my_background

[random name]
Name = application
Image =image_name
imageHeight= 80%
ImAgewidth=20px
CommAnd = /usr/bin/bla

[2ndApplication]
Name=2ndapplication
Command = a_command
[2ndApplication]
Name=3rdapplication
cOmmAnd= my_command`)
var invalidInifileMultipleGlobal []byte = []byte(
	`[Global]
background=

[global]`)
var invalidInifileUnknownKeys []byte = []byte(
	`[app]
Name= application
Command =command
invalidKey = value
`)
var invalidIniNoEntries []byte = []byte(
	`[Global]
Background = my_background`)

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

// TestParseConfigFile tests parseConfigFile with a mock config []byte (in place
// of a real file)
func TestParseConfigFile(t *testing.T) {
	validIniFileConfig := config{
		settings: &settings{
			Background: "my_background",
		},
		entries: []entry{
			{
				Name:        "application",
				Image:       "image_name",
				ImageHeight: "80%",
				ImageWidth:  "20px",
				Command:     "/usr/bin/bla",
			},
			{
				Name:    "2ndapplication",
				Command: "a_command",
			},
			{
				Name:    "3rdapplication",
				Command: "my_command",
			},
		},
	}
	tests := []struct {
		name      string
		input     []byte
		want      *config
		wantError bool
	}{
		{"valid INI", validIniFile, &validIniFileConfig, false},
		{`invalid INI with multiple "Global" sections`, invalidInifileMultipleGlobal, nil, true},
		{"invalid INI with undefined keys", invalidInifileUnknownKeys, nil, true},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			ini, err := ini.LoadSources(
				ini.LoadOptions{
					AllowNonUniqueSections: true,
					Insensitive:            true,
				},
				tests[i].input)
			if err != nil {
				t.Fatal(err)
			}
			result, err := parseConfigFile(ini)
			if tests[i].wantError && err == nil {
				t.Errorf("should fail but got %+v", result)
			} else if tests[i].wantError {
				return
			} else if err != nil {
				t.Errorf("failed with %q", err)
			} else {
				if *result.settings != *tests[i].want.settings {
					t.Errorf("want settings %+v but got %+v", *tests[i].want.settings, *result.settings)
				}
				if !compareSlices(result.entries, tests[i].want.entries) {
					t.Errorf("want entries %+v but got %+v", tests[i].want.entries, result.entries)
				}
			}
		})
	}
}

func TestValidateMeasurement(t *testing.T) {
	tests := []struct {
		name      string
		input     entryMeasurement
		wantError bool
	}{
		{"%52", entryMeasurement("%52"), false},
		{"%5", entryMeasurement("%5"), false},
		{"%100", entryMeasurement("%100"), false},
		{"52%", entryMeasurement("52%"), false},
		{"5%", entryMeasurement("5%"), false},
		{"100%", entryMeasurement("100%"), false},
		{"2502px", entryMeasurement("2502px"), false},
		{"1px", entryMeasurement("1px"), false},
		{"", entryMeasurement(""), false},
		{"20p", entryMeasurement("20p"), true},
		{"55", entryMeasurement("55"), true},
		{"%155", entryMeasurement("%155"), true},
		{"155%", entryMeasurement("155%"), true},
		{"%-15", entryMeasurement("%-15"), true},
		{"-15%", entryMeasurement("-15%"), true},
		{"%0", entryMeasurement("%0"), true},
		{"0%", entryMeasurement("0%"), true},
		{"word", entryMeasurement("word"), true},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {

			err := tests[i].input.validate()
			if tests[i].wantError && err == nil {
				t.Errorf("measurement %q should not parse correctly but did", tests[i].input)
			} else if !tests[i].wantError && err != nil {
				t.Errorf("measurement %q should parse but returned error %s", tests[i].input, err)
			}
		})
	}
}

func TestValidateEntries(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantError bool
	}{
		{"valid INI", validIniFile, false},
		{"INI without any app entries", invalidIniNoEntries, true},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {

			ini, err := ini.LoadSources(
				ini.LoadOptions{
					AllowNonUniqueSections: true,
					Insensitive:            true,
				},
				tests[i].input)
			if err != nil {
				t.Fatal(err)
			}

			result, _ := parseConfigFile(ini)
			// remove images from the entries, they should be tested separately
			for i := range result.entries {
				result.entries[i].Image = ""
			}

			err = validateEntries(result.entries)

			if tests[i].wantError && err == nil {
				t.Errorf("validating %+v should fail but didn't", result.entries)
			} else if !tests[i].wantError && err != nil {
				t.Errorf("%+v should validate but produced error %s", result.entries, err)
			}
		})
	}
}

func TestValidateImage(t *testing.T) {
	tests := []struct {
		name       string
		imgContent []byte
		wantError  bool
	}{
		{"valid png file", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, false},
		{"valid jpeg file", []byte{0xFF, 0xD8, 0xFF, 0xE0}, false},
		// {"valid bmp file", []byte{0x42, 0x4D, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x36, 0x00, 0x00, 0x00}, false},
		{"invalid file", []byte("not an image"), true},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "img-file-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.Write(tests[i].imgContent); err != nil {
				t.Fatal(err)
			}

			img := entryImage(tmpFile.Name())
			err = img.validate()

			if (err != nil) != tests[i].wantError {
				t.Errorf("wantError: %v but produced error %v", tests[i].wantError, err)
			}
		})
	}
}
