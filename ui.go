package main

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type app struct {
	mut        *sync.Mutex
	settings   *settings
	entries    []entry
	images     []*entryImg
	scrW, scrH int
}

type entryImg struct {
	img           *ebiten.Image
	name          string
	height, width int
	titleWidth    float64
	imgTitPad     int
	err           *error
}

func (a *app) Layout(winW, winH int) (int, int) {
	if (winW != a.scrW) || (winH != a.scrH) {
		a.scrW, a.scrH = winW, winH
		// TODO: handle error possibly thrown by this:
		a.generateEntryImgs()
	}
	return winW, winH
}

func (a *app) Update() error {
	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	return
}

// getTitleDrawX returns the correct origin point on the x axis for drawing the
// title
func getTitleDrawX(eImg *entryImg) float64 {
	canvasWidth := float64(eImg.width)
	var x float64
	if eImg.titleWidth > canvasWidth {
		x = 0 - ((eImg.titleWidth - canvasWidth) / 2)
	} else {
		x = (canvasWidth - eImg.titleWidth) / 2
	}
	return x
}

// getIconDimensions takes height, compares it with the maximal height available
// on eImg and uses the smaller value to compute width; it returnes the correct
// height and width
func getIconDimensions(height float64, eImg *entryImg) (float64, float64) {
	maxHeight := float64(eImg.img.Bounds().Size().Y)
	maxWidth := float64(eImg.img.Bounds().Size().X)
	aspectRatio := maxWidth / maxHeight
	if height > maxHeight {
		height = maxHeight
	}
	width := height * aspectRatio
	return width, height
}

// drawIconOnEntryImg calculates the dimensions and position for the icon in
// e and draws it onto entryImg
func (a *app) drawIconOnEntryImg(e *entry, eImg *entryImg) {
	icon := ebiten.NewImageFromImage(*e.Icon.image)
	iconOpt := &ebiten.DrawImageOptions{}
	iconWidth, iconHeight := getIconDimensions(e.IconHeight.abs, eImg)
	iconOpt.GeoM.Scale(iconWidth, iconHeight)

	// - x: width of the canvas minus width of the icon, the resulting difference
	//      divided by two
	//      (canvasWidth - iconWidth) / 2
	// - y: available height for the image - image height, the resulting
	//      difference divided by two
	//      (canvasHeight - imageTitlePadding - titleHeight - imageHeight) / 2
	iconX := (float64(eImg.width) - iconWidth) / 2
	a.mut.Lock()
	iconY := float64(eImg.height) - a.settings.ImageTitlePadding.abs
	a.mut.Unlock()
	iconOpt.GeoM.Translate(iconX, iconY)
	icon.DrawImage(eImg.img, iconOpt)
}

// drawTitleOnEntryImg calculates the dimensions and position for the title in e
// and draws it onto entryImg
func (a *app) drawTitleOnEntryImg(e *entry, eImg *entryImg) {
	// - x: width of the canvas minus width of the title, the resulting
	//      difference divided by two
	//      (canvasWidth - titleWidth) / 2
	// - y: height of the canvas minus the font height
	// the title may overlap lengthwise, in that case:
	// -x: the x origin point of the canvas minus the difference of the width of
	//     the title and the width of the canvas divided by two
	//     canvasWidth - ((titleWidth - canvasWidth) / 2)
	titleX := getTitleDrawX(eImg)
	a.mut.Lock() // accessing app
	titleY := float64(eImg.height) - a.settings.Font.height
	a.mut.Unlock()
	titleOpt := &text.DrawOptions{}
	titleOpt.GeoM.Translate(titleX, titleY)
	a.mut.Lock() // accessing app
	text.Draw(eImg.img, eImg.name, a.settings.Font.face, titleOpt)
	a.mut.Unlock()
}

// getEntryImg creates an entryImage for an entry reading e and respecting maxHeight
// and fontHeight
func (a *app) getEntryImg(e *entry, eImg *entryImg, wg *sync.WaitGroup) {
	defer wg.Done()
	eImg.img = ebiten.NewImage(eImg.width, eImg.height)
	a.drawIconOnEntryImg(e, eImg)
	a.drawTitleOnEntryImg(e, eImg)
	a.mut.Lock()
	a.images = append(a.images, eImg)
	a.mut.Unlock()
}

// newEntryImg initializes a new entryImg instance reading relevant values from
// a and then returns a pointer to it
func (a *app) newEntryImg(title string) *entryImg {
	img := new(entryImg)
	img.name = title
	img.height = a.scrH - int(a.settings.TopPadding.abs) - int(a.settings.BottomPadding.abs)
	img.width = a.scrW
	img.imgTitPad = int(a.settings.ImageTitlePadding.abs)
	img.titleWidth, _ = text.Measure(title, a.settings.Font.face, 0)
	return img
}

// generateEntryImgs creates drawable images from all entries in the config; the
// dimensions are calculated by taking into account size and padding options
// from the settings
func (a *app) generateEntryImgs() error {
	if err := a.validateDimensions(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	for i := range a.entries {
		eImg := a.newEntryImg(a.entries[i].Name)
		a.images = append(a.images, eImg)
		wg.Add(1)
		go a.getEntryImg(&a.entries[i], eImg, &wg)
	}
	wg.Wait()
	return nil
}

// validateDimensions checks if the provided dimensions leave enough room to
// actually draw the required elements of each entry
func (a *app) validateDimensions() error {
	imgHeight := float64(a.scrH) - a.settings.Font.height - a.settings.TopPadding.abs - a.settings.BottomPadding.abs - a.settings.ImageTitlePadding.abs
	if int(math.Round(imgHeight+0.5)) < 1 {
		return &dimensionError{}
	} else {
		return nil
	}
}

// init initializes the app struct so all values required for execution are
// available
func (a *app) init() error {
	// prime the font field
	return nil
}

// runGui creates a GUI using the settings taken from conf
func runGui(conf *config) {

}
