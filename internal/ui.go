package main

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type dimensions[T int | float64] struct {
	width, height T
}

type app struct {
	mut         *sync.Mutex
	settings    *settings
	entries     []entry
	images      []*entryImg
	screenDim   dimensions[int]
	entryImgDim dimensions[float64]
}

// entryImg contains all information about the graphical representation of an
// entry. The actual ebiten image is embedded in entryImg.img. It will be drawn
// to the screen with its origin point at [TopPadding, LeftPadding] set in
// app.settings. The ebiten image contains a properly aligned icon and title as
// specified in the entry that entryImg was created from.
type entryImg struct {
	img        *ebiten.Image
	name       string
	dim        dimensions[float64]
	titleDim   dimensions[float64]
	iconTitPad int
	err        *error
}

func (a *app) Layout(winW, winH int) (int, int) {
	if (winW != a.screenDim.width) || (winH != a.screenDim.height) {
		// save screen dimensions in app
		a.screenDim.width, a.screenDim.height = winW, winH
		// save entryImg dimensions in app
		a.entryImgDim.height = float64(winH) - a.settings.TopPadding.abs - a.settings.BottomPadding.abs
		a.entryImgDim.width = float64(winW) - a.settings.LeftPadding.abs - a.settings.RightPadding.abs
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
	canvasWidth := eImg.dim.width
	var x float64
	if eImg.titleDim.width > canvasWidth {
		x = 0 - ((eImg.titleDim.width - canvasWidth) / 2)
	} else {
		x = (canvasWidth - eImg.titleDim.width) / 2
	}
	return x
}

// getIconDimensions takes height, compares it with the maximal height available
// on eImg and uses the smaller value to compute width; it returnes the correct
// height and width
func getIconDimensions(e *entry, eImg *entryImg) (float64, float64) {
	maxWidth := float64(eImg.img.Bounds().Size().X)
	maxHeight := float64(eImg.img.Bounds().Size().Y)
	var height float64

	if e.Icon.isVector {
		height = min(maxHeight, e.IconHeight.abs)
	} else {
		height = min(maxHeight, e.IconHeight.abs, float64(e.Icon.dim.height))
	}

	// calculate width matching aspect ratio
	width := e.Icon.calculateWithAspectRatio(height, true)
	// if width is too big, adapt height instead using maxWidth
	if width > maxWidth {
		width = maxWidth
		height = e.Icon.calculateWithAspectRatio(width, false)
	}
	return width, height
}

// updateMeasurements should be called whenever there is a layout change. It
// will recalculate all measurement strings.
func (a *app) updateMeasurements() {
	height := float64(a.screenDim.height)
	var wg sync.WaitGroup
	for i := range a.entries {
		wg.Add(1)
		go a.entries[i].IconHeight.init(height, &wg)
	}
	// initiate all padding fields (without reflect for performance reasons)
	wg.Add(5)
	a.settings.IconTitlePadding.init(height, &wg)
	a.settings.TopPadding.init(height, &wg)
	a.settings.BottomPadding.init(height, &wg)
	a.settings.LeftPadding.init(height, &wg)
	a.settings.RightPadding.init(height, &wg)
	wg.Wait()
}

// drawIconOnEntryImg calculates the dimensions and position for the icon in
// e and draws it onto entryImg
func (a *app) drawIconOnEntryImg(e *entry, eImg *entryImg) {
	icon := ebiten.NewImageFromImage(e.Icon.image)
	iconOpt := &ebiten.DrawImageOptions{}
	iconWidth, iconHeight := getIconDimensions(e, eImg)
	iconOpt.GeoM.Scale(iconWidth, iconHeight)

	// - x: width of the canvas minus width of the icon, the resulting difference
	//      divided by two
	//      (canvasWidth - iconWidth) / 2
	// - y: available height for the image - image height, the resulting
	//      difference divided by two
	//      (canvasHeight - imageTitlePadding - titleHeight - imageHeight) / 2
	iconX := (float64(eImg.dim.width) - iconWidth) / 2
	a.mut.Lock()
	iconY := float64(eImg.dim.height) - a.settings.ImageTitlePadding.abs
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
	titleY := eImg.dim.height - a.settings.Font.height
	a.mut.Unlock()
	titleOpt := &text.DrawOptions{}
	titleOpt.GeoM.Translate(titleX, titleY)
	a.mut.Lock() // accessing app
	text.Draw(eImg.img, eImg.name, a.settings.Font.face, titleOpt)
	a.mut.Unlock()
}

// getEntryEbitenImg creates an ebiten image, draws the icon and the title onto
// it and stores it in eImg.img. eImg is then appended to a.images.
func (a *app) getEntryEbitenImg(e *entry, eImg *entryImg, wg *sync.WaitGroup) {
	defer wg.Done()
	eImg.img = ebiten.NewImage(int(eImg.dim.width), int(eImg.dim.height))
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
	img.dim.height = a.entryImgDim.height
	img.dim.width = a.entryImgDim.width
	img.iconTitPad = int(a.settings.IconTitlePadding.abs)
	img.titleDim.width, _ = text.Measure(title, a.settings.Font.face, 0)
	img.titleDim.height = a.settings.Font.height
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
		go a.getEntryEbitenImg(&a.entries[i], eImg, &wg)
	}
	wg.Wait()
	// TODO: sort a.images
	return nil
}

// validateDimensions checks if the provided dimensions leave enough room to
// actually draw the required elements of each entry
func (a *app) validateDimensions() error {
	imgHeight := float64(a.screenDim.height) - a.settings.Font.height - a.settings.TopPadding.abs - a.settings.BottomPadding.abs - a.settings.IconTitlePadding.abs
	if int(math.Round(imgHeight+0.5)) < 1 {
		return &dimensionError{}
	} else {
		return nil
	}
}

// init initializes the app struct so all values required for execution are
// available
func (a *app) init() error {
	// load image fields
	for i := range a.entries {
		if err := a.entries[i].Icon.init(); err != nil {
			return err
		}
	}

	// load font
	if err := a.settings.Font.init(float64(a.settings.FontSize)); err != nil {
		return err
	}

	return nil
}

// getApp returns a new instance of app and makes sure there are no nil pointer
func getApp(conf *config) *app {
	a := new(app)
	a.mut = &sync.Mutex{}
	a.settings = conf.settings
	a.entries = conf.entries
	return a
}

// runGui creates a GUI using the settings taken from conf
func runGui(conf *config) error {
	conf, err := getConfig()
	if err != nil {
		return err
	}

	a := getApp(conf)
	if err := a.init(); err != nil {
		return err
	}

	ebiten.SetWindowTitle("flaunder")
	ebiten.SetFullscreen(true)
	if err := ebiten.RunGame(a); err != nil {
		return err
	}
	return nil
}
