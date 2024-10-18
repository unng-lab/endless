package ui

import (
	"bytes"
	"image/color"
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/text/language"

	"github.com/ebitenui/ebitenui"
	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"

	"github/unng-lab/madfarmer/internal/camera"
)

type UIEngine struct {
	ebitenui.UI
	camera  *camera.Camera
	clicked bool
}

func New(camera *camera.Camera) *UIEngine {
	var ui UIEngine
	ui.camera = camera

	// construct a new container that serves as the root of the UI hierarchy
	rootContainer := widget.NewContainer(
		// the container will use an anchor layout to layout its single child widget
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			//Define number of columns in the grid
			widget.GridLayoutOpts.Columns(2),
			//Define how much padding to inset the child content
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(0)),
			//Define how far apart the rows and columns should be
			widget.GridLayoutOpts.Spacing(0, 0),
			//Define how to stretch the rows and columns. Note it is required to
			//specify the Stretch for each row and column.
			widget.GridLayoutOpts.Stretch([]bool{true, true}, []bool{true, true}),
		)),
	)

	rootContainer.AddChild(ui.leftTopContainer())
	rootContainer.AddChild(ui.rightTopContainer())
	rootContainer.AddChild(ui.leftBottomContainer())
	rootContainer.AddChild(ui.rightBottomContainer())
	ui.Container = rootContainer
	return &ui
}

func loadFont(size float64) (text.Face, error) {
	ttfFont, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, err
	}

	return &text.GoTextFace{
		Size:      size,
		Source:    ttfFont,
		Language:  language.English,
		Direction: text.DirectionLeftToRight,
	}, nil
}

func loadButtonImage() (*widget.ButtonImage, error) {
	idle := e_image.NewNineSliceColor(color.NRGBA{R: 170, G: 170, B: 180, A: 255})
	hover := e_image.NewNineSliceColor(color.NRGBA{R: 130, G: 130, B: 150, A: 255})
	pressed := e_image.NewNineSliceColor(color.NRGBA{R: 100, G: 100, B: 120, A: 255})

	return &widget.ButtonImage{
		Idle:    idle,
		Hover:   hover,
		Pressed: pressed,
	}, nil
}

func (ui *UIEngine) Clicked() bool {
	return ui.clicked
}

func newBut(i int) *widget.Button {
	// load images for button states: idle, hover, and pressed
	buttonImage, _ := loadButtonImage()

	// construct a button
	button := widget.NewButton(
		// set general widget options
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionEnd,
			}),
		),

		// specify the images to use
		widget.ButtonOpts.Image(buttonImage),

		// add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			slog.Info("Clicked")
		}),
	)
	return button
}
