package ui

import (
	"github.com/ebitenui/ebitenui/widget"
)

func (ui *UIEngine) leftTopContainer() *widget.Container {
	innerContainer1 := widget.NewContainer(
		//widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.NRGBA{0, 0, 0, 255})),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionStart,
				VerticalPosition:   0,
			}),
		),
	)
	return innerContainer1
}
