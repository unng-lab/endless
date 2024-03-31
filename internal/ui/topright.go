package ui

import (
	"github.com/ebitenui/ebitenui/widget"
)

func (ui *UIEngine) rightTopContainer() *widget.Container {
	innerContainer2 := widget.NewContainer(
		//widget.ContainerOpts.BackgroundImage(e_image.NewNineSliceColor(color.White)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionEnd,
				VerticalPosition:   0,
			}),
		),
	)
	return innerContainer2
}
