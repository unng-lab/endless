package ui

import (
	"log/slog"
	"strconv"

	"github.com/ebitenui/ebitenui/widget"
)

func (ui *UIEngine) rightBottomContainer() *widget.Container {
	innerContainer4 := widget.NewContainer(
		//widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{0, 255, 255, 255})),
		// the container will use an anchor layout to layout its single child widget
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Inverted(true),
			widget.RowLayoutOpts.Spacing(5),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				MaxHeight:          200,
				VerticalPosition:   widget.GridLayoutPositionEnd,
				HorizontalPosition: widget.GridLayoutPositionEnd,
			}),
		),
	)

	//buttonContainer := widget.NewContainer(
	//	widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{0, 255, 255, 255})),
	//	widget.ContainerOpts.Layout(widget.NewRowLayout()),
	//	widget.ContainerOpts.WidgetOpts(
	//		widget.WidgetOpts.LayoutData(widget.GridLayoutData{
	//			VerticalPosition:   widget.GridLayoutPositionEnd,
	//			HorizontalPosition: widget.GridLayoutPositionEnd,
	//		}),
	//	),
	//)
	//
	//buttonContainer1 := widget.NewContainer(
	//	widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.White)),
	//	widget.ContainerOpts.Layout(widget.NewRowLayout()),
	//	widget.ContainerOpts.WidgetOpts(
	//		widget.WidgetOpts.LayoutData(widget.GridLayoutData{
	//			VerticalPosition:   widget.GridLayoutPositionEnd,
	//			HorizontalPosition: widget.GridLayoutPositionEnd,
	//		}),
	//	),
	//)

	for i := range 9 {
		innerContainer4.AddChild(newButBUT(i))
	}

	//innerContainer4.AddChild(buttonContainer)
	//innerContainer4.AddChild(buttonContainer1)

	return innerContainer4
}

func newButBUT(i int) *widget.Button {
	// load images for button states: idle, hover, and pressed
	buttonImage, _ := loadButtonImage()

	// construct a button
	button := widget.NewButton(
		// set general widget options
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionEnd,
				Stretch:  false,
			}),
		),

		// specify the images to use
		widget.ButtonOpts.Image(buttonImage),

		// add a handler that reacts to clicking the button
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			slog.Info("Clicked " + strconv.Itoa(i))
		}),
	)
	return button
}
