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
TopPadding = 30%
BottomPadding = 34px
Leftpadding=1%
rightpadding=200px

[random name]
Name = application
Icon =image_name
icOnheight= 80%
CommAnd = /usr/bin/bla

[2ndApplication]
Name=2ndapplication
Command = a_command
[2ndApplication]
Name=3rdapplication
cOmmAnd= my_command`)
var validIniFileNoFilenames []byte = []byte(
	`[Global]
Background = 

[random name]
Name = application
Icon =
ICONHeight= 80%
CommAnd =

[2ndApplication]
Name=2ndapplication
Command =
[2ndApplication]
Name=3rdapplication
cOmmAnd=`)
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
var invalidIniFileNoEntries []byte = []byte(
	`[Global]
Background = my_background`)
var invalidIniFileSyntaxErrors []byte = []byte(
	`[Global]
background =

[app]
Icon =
Command =
IconHeight= invalid`)

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
			Background: icon{
				path:  "my_background",
				image: nil},
			TopPadding: measurement{
				value: "30%",
				abs:   0},
			BottomPadding: measurement{
				value: "34px",
				abs:   0},
			LeftPadding: measurement{
				value: "1%",
				abs:   0},
			RightPadding: measurement{
				value: "200px",
				abs:   0},
		},
		entries: []entry{
			{
				Name: "application",
				Icon: icon{
					path:  "image_name",
					image: nil,
				},
				IconHeight: measurement{
					value: "80%",
					abs:   0,
				},
				Command: "/usr/bin/bla",
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

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantError bool
	}{
		{"valid INI", validIniFileNoFilenames, false},
		{"invalid INI with fields that shouldn't validate", invalidIniFileSyntaxErrors, true},
		{"INI without any app entries", invalidIniFileNoEntries, true},
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

			conf, err := parseConfigFile(ini)
			if err != nil {
				t.Fatal(err)
			}

			err = validateConfig(conf)

			if tests[i].wantError && err == nil {
				t.Errorf("validating %+v should fail but didn't", conf)
			} else if !tests[i].wantError && err != nil {
				t.Errorf("%+v should validate but produced error %s", conf, err)
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

			img := icon{path: tmpFile.Name()}
			err = img.validate()

			if (err != nil) != tests[i].wantError {
				t.Errorf("wantError: %v but produced error %v", tests[i].wantError, err)
			}
		})
	}
}
