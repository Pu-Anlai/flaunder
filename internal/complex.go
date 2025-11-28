package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strconv"
	"sync"

	"github.com/h2non/filetype"
	svg "github.com/h2non/go-is-svg"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

const dpi float64 = 96

type complexOption interface {
	setBaseField(string)
}

type icon struct {
	valid     bool
	path      string
	image     image.Image
	dim       dimensions
	isVector  bool
	vecCanvas *canvas.Canvas
}

type font struct {
	valid  bool
	path   string
	height float64
	face   *text.GoTextFace
}

type measurement struct {
	valid bool
	value string
	abs   float64
}

func (i *icon) setBaseField(v string) {
	i.path = v
}

// validate returns nil if i points to a supported image file, otherwise it
// returns an appropriate error
func (i *icon) validate() error {
	i.valid = true
	// not providing an image is allowed:
	if i.path == "" {
		return nil
	}

	buf, err := os.ReadFile(i.path)
	if err != nil {
		return &fileAccessError{path: i.path}
	}

	if svg.IsSVG(buf) {
		i.isVector = true
		return nil
	} else if filetype.IsImage(buf) {
		return nil
	} else {
		return &fileAccessError{path: i.path, fileType: "image"}
	}
}

func (i *icon) init() error {
	if !i.valid {
		panic(fmt.Sprintf("attempted to run init on unvalidated %+v", *i))
	}

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
	} else {
		img, ft, err := image.Decode(f)
		if err != nil {
			return &fileDecodeError{path: i.path, fileType: ft}
		}
		i.image = img
		bounds := img.Bounds()
		i.dim = dimensions{width: bounds.Dx(), height: bounds.Dy()}
	}
	return nil
}

// ensureRendered makes sure that an underlying svg file is rendered properly
// given height and present in field i.image
func (i *icon) ensureRendered(height float64) {
	if !i.isVector {
		return
	}
	width := i.calculateWithAspectRatio(height, true)
	canv := canvas.New(width, height)
	scaleX, scaleY := width/i.vecCanvas.W, height/i.vecCanvas.H

	i.vecCanvas.RenderViewTo(canv, canvas.Identity.Scale(scaleX, scaleY))
	i.image = rasterizer.Draw(canv, canvas.DPI(dpi), canvas.DefaultColorSpace)
}

// calculateWithAspectRatio uses one side (base) to calculate and return the
// other side maintaining the icon i's aspect ratio. if isHeight is true base is
// considered to be the height, otherwise it is considered the width
func (i *icon) calculateWithAspectRatio(base float64, isHeight bool) float64 {
	if i.isVector {
		if isHeight {
			return base * (i.vecCanvas.W / i.vecCanvas.H)
		} else {
			return base * (i.vecCanvas.H / i.vecCanvas.W)
		}
	} else {
		if isHeight {
			return base * (float64(i.dim.width) / float64(i.dim.height))
		} else {
			return base * (float64(i.dim.height) / float64(i.dim.width))
		}
	}
}

func (f *font) init(fontSize float64) error {
	if !f.valid {
		panic(fmt.Sprintf("attempted to run init on unvalidated %+v", *f))
	}

	var r io.Reader
	var fData []byte
	if f.path != "" {
		// this should be safe as the path has already been validated, even if
		// it hasn't, we can catch the error in the next step
		fData, _ = os.ReadFile(f.path)
	} else {
		fData = fbFont
	}
	r = bytes.NewReader(fData)

	fSource, err := text.NewGoTextFaceSource(r)
	if err != nil {
		// unlikely to trigger as the filetype package should have confirmed
		// this to be a valid font
		// if the fallback font is used, we know this to be a valid font
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
	f.valid = true
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
	if !m.valid {
		panic(fmt.Sprintf("attempted to run init on unvalidated %+v", *m))
	}

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
	m.valid = true
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
