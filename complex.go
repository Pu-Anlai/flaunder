package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strconv"

	"github.com/h2non/filetype"
	svg "github.com/h2non/go-is-svg"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type complexOption interface {
	setBaseField(string)
}

type icon struct {
	path  string
	image *image.Image
}

type font struct {
	path   string
	height float64
	face   *text.GoTextFace
}

type measurement struct {
	value string
	abs   float64
}

func (i *icon) setBaseField(v string) {
	i.path = v
}

// validate returns nil if i points to a supported image file, otherwise it
// returns an appropriate error
func (i *icon) validate() error {
	// not providing an image is allowed:
	if i.path == "" {
		return nil
	}

	buf, err := os.ReadFile(i.path)
	if err != nil {
		return &fileAccessError{path: i.path}
	}

	if !filetype.IsImage(buf) || svg.IsSVG(buf) {
		return &fileAccessError{path: i.path, fileType: "image"}
	}
	return nil
}

func (f *font) init(a *app) error {
	var r io.Reader
	if a.settings.Font.path != "" {
		// this should be safe as the path has already been validated, even if
		// it doesn't, we can catch the error in the next step
		fData, _ := os.ReadFile(a.settings.Font.path)
		r = bytes.NewReader(fData)
	} else {
		r = bytes.NewReader(fonts.MPlus1pRegular_ttf)
	}

	fSource, err := text.NewGoTextFaceSource(r)
	if err != nil {
		// unlikely to trigger as the filetype package should have confirmed
		// this to be a valid font
		return err
	}

	size := float64(a.settings.FontSize)
	if size == 0 {
		size = 18
	}

	a.settings.Font.face = &text.GoTextFace{
		Source: fSource,
		Size:   size,
	}
	metrics := a.settings.Font.face.Metrics()
	a.settings.Font.height = metrics.HAscent + metrics.HDescent

	return nil
}

func (f *font) setBaseField(v string) {
	f.path = v
}

// validate checks if path points to a valid font file
func (f *font) validate() error {
	// not providing a font is allowed
	if f.path == "" {
		return nil
	}

	fontFile, err := os.ReadFile(f.path)
	if err != nil {
		return &fileAccessError{
			path: f.path,
		}
	}
	if !filetype.IsFont(fontFile) {
		return &fileAccessError{
			path:     f.path,
			fileType: "font",
		}
	}

	return nil
}

func (m *measurement) init(rel float64) {
	if m.value[len(m.value)-1:] == "%" {
		m.abs, _ = strconv.ParseFloat(m.value[:len(m.value)-1], 10)
	} else if m.value[len(m.value)-2:] == "px" {
		m.abs, _ = strconv.ParseFloat(m.value[:len(m.value)-2], 10)
	}
}

func (m *measurement) setBaseField(v string) {
	m.value = v
}

// validate makes sure measurement follows one of the allowed patterns
func (m *measurement) validate() error {
	if len(m.value) == 1 {
		return &iniParseError{value: m.value}
	}

	// no value means 100%
	if m.value == "" {
		m.value = "100%"
		return nil
	} else if m.value[len(m.value)-1:] == "%" {
		uI, _ := strconv.ParseUint(m.value[:len(m.value)-1], 10, 32)
		if uI == 0 || uI > 100 {
			return &iniParseError{value: m.value}
		}
	} else if m.value[len(m.value)-2:] == "px" {
		uI, _ := strconv.ParseUint(m.value[:len(m.value)-2], 10, 32)
		if uI == 0 {
			return &iniParseError{value: m.value}
		}
	} else {
		return &iniParseError{value: m.value}
	}

	return nil
}

