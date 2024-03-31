package ui

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

func (ui *UIEngine) leftBottomContainer() *widget.Container {
	// load images for button states: idle, hover, and pressed
	buttonImage, _ := loadButtonImage()

	// load text font
	face, _ := loadFont(16)

	innerContainer3 := widget.NewContainer(
		//widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 0, 255, 255})),
		// the container will use an anchor layout to layout its single child widget
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			//The widget in this cell has a MaxHeight and MaxWidth less than the
			//Size of the grid cell so it will use the Position fields below to
			//Determine where the widget should be displayed within that grid cell.
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				//HorizontalPosition: widget.GridLayoutPositionCenter,
				VerticalPosition: widget.GridLayoutPositionEnd,
				MaxHeight:        100,
			}),
		),
	)

	// Create the first tab
	// A TabBookTab is a labelled container. The text here is what will show up in the tab button
	tabRed := widget.NewTabBookTab("Red Tab",
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{255, 0, 0, 255})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	redBtn := widget.NewText(
		widget.TextOpts.Text("Red Tab Button", face, color.White),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	tabRed.AddChild(redBtn)

	tabGreen := widget.NewTabBookTab("Green Tab",
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 255, 0, 0xff})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	greenBtn := widget.NewText(
		widget.TextOpts.Text("Green Tab Button\nThis is configured as the initial tab.", face, color.Black),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	tabGreen.AddChild(greenBtn)

	tabBlue := widget.NewTabBookTab("Blue Tab",
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 0, 255, 0xff})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
		)),
	)
	blueBtn1 := widget.NewText(
		widget.TextOpts.Text("Blue Tab Button 1", face, color.White),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	)
	tabBlue.AddChild(blueBtn1)
	blueBtn2 := widget.NewText(
		widget.TextOpts.Text("Blue Tab Button 2", face, color.White),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Position: widget.RowLayoutPositionCenter,
		})),
	)
	tabBlue.AddChild(blueBtn2)

	tabDisabled := widget.NewTabBookTab("Disabled Tab",
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{R: 80, G: 80, B: 140, A: 255})),
	)
	tabDisabled.Disabled = true

	tabBook := widget.NewTabBook(
		widget.TabBookOpts.TabButtonImage(buttonImage),
		widget.TabBookOpts.TabButtonText(face, &widget.ButtonTextColor{Idle: color.White, Disabled: color.White}),
		widget.TabBookOpts.TabButtonSpacing(0),
		widget.TabBookOpts.ContainerOpts(
			widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				StretchHorizontal:  true,
				StretchVertical:    true,
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			),
		),
		widget.TabBookOpts.TabButtonOpts(
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(98, 0)),
		),
		widget.TabBookOpts.Tabs(tabDisabled, tabRed, tabGreen, tabBlue),
		//	widget.TabBookOpts.InitialTab(tabGreen),
	)
	// add the tabBook as a child of the container
	innerContainer3.AddChild(tabBook)

	return innerContainer3
}
