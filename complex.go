package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strconv"
	"sync"

	"github.com/h2non/filetype"
	svg "github.com/h2non/go-is-svg"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

const dpi float64 = 96

type complexOption interface {
	setBaseField(string)
}

type icon struct {
	path        string
	image       image.Image
	dim         dimensions
	aspectRatio float64
	isVector    bool
	vecCanvas   *canvas.Canvas
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

	if filetype.IsImage(buf) {
		return nil
	} else if svg.IsSVG(buf) {
		i.isVector = true
		return nil
	} else {
		return &fileAccessError{path: i.path, fileType: "image"}
	}
}

func (i *icon) init() error {
	f, err := os.Open(i.path)
	if err != nil {
		return &fileAccessError{path: i.path, fileType: "image"}
	}
	defer f.Close()

	if i.isVector {
		svgCanvas, err := canvas.ParseSVG(f)
		if err != nil {
			return &fileDecodeError{path: i.path, fileType: "svg"}
		}
		i.vecCanvas = svgCanvas
		i.aspectRatio = svgCanvas.H / svgCanvas.W
	} else {
		img, ft, err := image.Decode(f)
		if err != nil {
			return &fileDecodeError{path: i.path, fileType: ft}
		}
		i.image = img
		bounds := img.Bounds()
		i.dim = dimensions{width: bounds.Dx(), height: bounds.Dy()}
		i.aspectRatio = float64(bounds.Dx()) / float64(bounds.Dy())
	}
	return nil
}

// ensureRendered makes sure that an underlying svg file is rendered properly
// given height and present in field i.image
func (i *icon) ensureRendered(height float64) {
	if !i.isVector {
		return
	}
	width := height * i.aspectRatio
	canv := canvas.New(width, height)
	scaleX, scaleY := width/i.vecCanvas.W, height/i.vecCanvas.H

	i.vecCanvas.RenderViewTo(canv, canvas.Identity.Scale(scaleX, scaleY))
	i.image = rasterizer.Draw(canv, canvas.DPI(dpi), canvas.DefaultColorSpace)
}

func (f *font) init(fontSize float64) error {
	var r io.Reader
	if f.path != "" {
		// this should be safe as the path has already been validated, even if
		// it doesn't, we can catch the error in the next step
		fData, _ := os.ReadFile(f.path)
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

	if fontSize == 0 {
		fontSize = 18
	}

	f.face = &text.GoTextFace{
		Source: fSource,
		Size:   fontSize,
	}
	metrics := f.face.Metrics()
	f.height = metrics.HAscent + metrics.HDescent

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

func (m *measurement) init(rel float64, wg *sync.WaitGroup) {
	if m.value[len(m.value)-1:] == "%" {
		parsed, _ := strconv.ParseFloat(m.value[:len(m.value)-1], 10)
		m.abs = rel * (parsed / 100)
	} else if m.value[len(m.value)-2:] == "px" {
		m.abs, _ = strconv.ParseFloat(m.value[:len(m.value)-2], 10)
	}
	wg.Done()
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
