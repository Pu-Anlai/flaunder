package flunder

import (
	"fmt"
	"image"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type dimensionError struct {
	entryName string
}

func (e *dimensionError) Error() string {
	if e.entryName == "" {
		return fmt.Sprintf("global settings do not leave enough space for drawing elements")
	} else {
		return fmt.Sprintf("settings for %q do not leave enough space for drawing all its elements", e.entryName)
	}
}

type dimensions[T int | float64] struct {
	width, height T
}

type background struct {
	image     *ebiten.Image
	imageOpts *ebiten.DrawImageOptions
}

type app struct {
	mut         *sync.Mutex
	settings    *settings
	entries     []entry
	images      []*entryImg
	screenDim   dimensions[int]
	entryImgDim dimensions[float64]
	bg          background
}

// entryImg contains all information about the graphical representation of an
// entry. The actual ebiten image is embedded in entryImg.img. It will be drawn
// to the screen with its origin point at [TopPadding, LeftPadding] set in
// app.settings. The ebiten image contains a properly aligned icon and title as
// specified in the entry that entryImg was created from.
type entryImg struct {
	img          *ebiten.Image
	name         string
	dim          dimensions[float64]
	titleDim     dimensions[float64]
	iconTitlePad int
	err          *error
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

}

// updateBg creates an ebiten image containing the background image, scales
// it to fit the screen (if BackgroundScale is true) and stores it in the app's
// background field
func (a *app) updateBg() {
	bg := &a.settings.Background // shorthand so things don't get too unwieldy

	if a.settings.BackgroundScale {
		var w, h int
		if (bg.dim.width / a.screenDim.width) > (bg.dim.height / a.screenDim.height) {
			w = a.screenDim.width
			h = w * (bg.dim.height / bg.dim.width)
		} else {
			h = a.screenDim.height
			w = h * (bg.dim.width / bg.dim.height)
		}
		a.bg.imageOpts.GeoM.Reset()
		scaleFactor := dimensionsToScaleFactor(a.bg.image, dimensions[float64]{float64(w), float64(h)})
		a.bg.imageOpts.GeoM.Scale(scaleFactor[0], scaleFactor[1])
	}
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

// dimensionsToScaleFactor calculates the scale factor to scale ebitImg to
// targetDim. It returns an array containing the width and height scale factor.
func dimensionsToScaleFactor(img image.Image, targetDim dimensions[float64]) [2]float64 {
	origSize := img.Bounds().Size()
	w := targetDim.width / float64(origSize.X)
	h := targetDim.height / float64(origSize.Y)
	return [2]float64{w, h}
}

// getTitleOriginPoint calculates the origin point for drawing the title onto
// eImg so that it is aligned centrally on the x axis and on top of the bottom
// padding on the y axis
func getTitleOriginPoint(eImg *entryImg) (x, y float64) {
	// - x: width of the canvas (eImg.img, the ebiten image) minus width of the
	// 	 	title, the resulting difference divided by two
	//      (canvasWidth - titleWidth) / 2
	// - y: height of the canvas minus the title height
	// 	 	(canvasHeight - titleHeight)
	x = (eImg.dim.width - eImg.titleDim.width) / 2
	y = eImg.dim.height - eImg.titleDim.height
	return x, y
}

// getIconDimensions takes height, compares it with the maximal height available
// on eImg and uses the smaller value to compute width; it returnes the correct
// height and width
func getIconDimensions(e *entry, eImg *entryImg) dimensions[float64] {
	maxWidth := float64(eImg.img.Bounds().Size().X)
	// we need to make sure that sufficient space is still available for the
	// title and the image title padding after drawing the icon
	maxHeight := float64(eImg.img.Bounds().Size().Y) - float64(eImg.iconTitlePad) - eImg.titleDim.height
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
	return dimensions[float64]{width, height}
}

// getBgOriginPoint calculates the origin point for drawing the background
func (a *app) getBgOriginPoint(canvasDim, bgDim dimensions[float64]) (x, y float64) {
	// x and y are both set to an origin point that should center the image
	x = (canvasDim.width - bgDim.width) / 2
	y = (canvasDim.height - bgDim.height) / 2
	return x, y
}

// getIconOriginPoint calculates the origin point for drawing the icon onto eImg
// so that it is aligned centrally on the x axis and center between top padding
// and icon-title padding on the y axis
func getIconOriginPoint(eImg *entryImg, iconDim dimensions[float64]) (x, y float64) {
	// - x: width of the canvas (the ebiten image) minus width of the icon, the resulting difference
	//      divided by two
	//      (canvasWidth - iconWidth) / 2
	// - y: height of the canvas (the ebiten image) minus height of the icon
	//      minus iconTitlePadding minus the height of the title, the difference
	//      divided by two - if the icon has its maximum height, y should be 0
	//      (canvasWidth - iconHeight - iconTitlePadding - titleHeight) / 2
	x = (eImg.dim.width - iconDim.width) / 2
	y = (eImg.dim.height - iconDim.height - float64(eImg.iconTitlePad) - eImg.titleDim.height) / 2
	return x, y
}

// drawIconOnEntryImg calculates the dimensions and position for the icon in
// e and draws it onto entryImg
func (a *app) drawIconOnEntryImg(e *entry, eImg *entryImg) {
	ebitIcon := ebiten.NewImageFromImage(e.Icon.image)
	iconOpt := &ebiten.DrawImageOptions{}
	iconDim := getIconDimensions(e, eImg)
	scaleFactor := dimensionsToScaleFactor(ebitIcon, iconDim)
	iconOpt.GeoM.Scale(scaleFactor[0], scaleFactor[1])

	iconX, iconY := getIconOriginPoint(eImg, iconDim)
	iconOpt.GeoM.Translate(iconX, iconY)
	ebitIcon.DrawImage(eImg.img, iconOpt)
}

// drawTitleOnEntryImg calculates the dimensions and position for the title in e
// and draws it onto entryImg
func (a *app) drawTitleOnEntryImg(eImg *entryImg) {
	titleX, titleY := getTitleOriginPoint(eImg)
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
	a.drawTitleOnEntryImg(eImg)
	a.mut.Lock()
	a.images = append(a.images, eImg)
	a.mut.Unlock()
}

// newEntryImg initializes a new entryImg instance reading relevant values from
// a and then returns a pointer to it
func (a *app) newEntryImg(title string) *entryImg {
	eImg := new(entryImg)
	eImg.name = title
	eImg.dim.height = a.entryImgDim.height
	eImg.dim.width = a.entryImgDim.width
	eImg.iconTitlePad = int(a.settings.IconTitlePadding.abs)
	eImg.titleDim.width, _ = text.Measure(title, a.settings.Font.face, 0)
	eImg.titleDim.height = a.settings.Font.height
	return eImg
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

	// set up background
	if err := a.settings.Background.init(); err != nil {
		return err
	}
	a.bg.image = ebiten.NewImageFromImage(a.settings.Background.image)
	a.bg.imageOpts = &ebiten.DrawImageOptions{}

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

// RunGui creates a GUI using the settings taken from conf
func RunGui(conf *config) error {
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
