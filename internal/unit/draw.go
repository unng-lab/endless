package unit

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"

	"github/unng-lab/madfarmer/internal/camera"
	"github/unng-lab/madfarmer/internal/geom"
)

func (u *Unit) Draw(screen *ebiten.Image, counter int) bool {
	defer u.DrawOptions.GeoM.Reset()
	u.DrawOptions.GeoM.Scale(
		u.Camera.ScaleFactor(),
		u.Camera.ScaleFactor(),
	)
	drawPoint := u.GetDrawPoint()
	u.DrawOptions.GeoM.Translate(drawPoint.X, drawPoint.Y)
	if u.Focused {
		screen.DrawImage(u.FocusedAnimation[counter%len(u.FocusedAnimation)], &u.DrawOptions)
		u.DrawTitle(screen)
	} else {
		screen.DrawImage(u.Animation[counter%len(u.Animation)], &u.DrawOptions)
	}

	if drawRect {
		u.drawRect(screen)
	}

	//ebitenutil.DebugPrintAt(
	//	screen,
	//	fmt.Sprintf(
	//		`posX: %0.2f,
	//posY: %0.2f,
	//name: %s,
	//uposX: %0.2f,
	//uposY: %0.2f`,
	//		u.DrawOptions.GeoM.Element(0, 2),
	//		u.DrawOptions.GeoM.Element(1, 2),
	//		u.Name,
	//		u.Position.X,
	//		u.Position.Y,
	//	),qqqq
	//	100,
	//	0,
	//)
	return true
}

func (u *Unit) DrawTitle(screen *ebiten.Image) {
	// Создаем шрифт
	fontFace := basicfont.Face7x13

	// Рисуем текст на экране
	text.DrawWithOptions(screen, u.Name, fontFace, &u.DrawOptions)
}

func (u *Unit) drawRect(screen *ebiten.Image) {
	leftAngle := u.GetDrawPoint()
	posX, posY := leftAngle.X, leftAngle.Y

	vector.StrokeLine(
		screen,
		float32(posX),
		float32(posY),
		float32(posX+u.Camera.TileSize()*u.SizeX),
		float32(posY),
		1,
		color.White,
		false,
	)
	vector.StrokeLine(
		screen,
		float32(posX+u.Camera.TileSize()*u.SizeX),
		float32(posY),
		float32(posX+u.Camera.TileSize()*u.SizeX),
		float32(posY+u.Camera.TileSize()*u.SizeY),
		1,
		color.White,
		false,
	)
	vector.StrokeLine(
		screen,
		float32(posX),
		float32(posY),
		float32(posX),
		float32(posY+u.Camera.TileSize()*u.SizeY),
		1,
		color.White,
		false,
	)
	vector.StrokeLine(
		screen,
		float32(posX),
		float32(posY+u.Camera.TileSize()*u.SizeY),
		float32(posX+u.Camera.TileSize()*u.SizeX),
		float32(posY+u.Camera.TileSize()*u.SizeY),
		1,
		color.White,
		false,
	)
}

func (u *Unit) GetDrawPoint() geom.Point {
	drawPoint := u.Camera.PointToCameraPixel(geom.Point{
		X: u.Position.X + u.PositionShiftX,
		Y: u.Position.Y + u.PositionShiftY,
	})
	return drawPoint
}

func (u *Unit) DrawPath(screen *ebiten.Image, camera camera.Camera) {
	if len(u.Pathing.Path) <= 1 {
		panic("path is empty")
	}
	u.Pathing.Path[len(u.Pathing.Path)-1] = geom.Pt(u.Position.X, u.Position.Y)

	for i := len(u.Pathing.Path) - 1; i > 0; i-- {
		if !camera.Coordinates.ContainsOR(u.Pathing.Path[i]) ||
			!camera.Coordinates.ContainsOR(u.Pathing.Path[i-1]) {
			start := camera.MiddleOfPointInRelativePixels(u.Pathing.Path[i])
			finish := camera.MiddleOfPointInRelativePixels(u.Pathing.Path[i-1])
			vector.StrokeLine(screen,
				float32(start.X),
				float32(start.Y),
				float32(finish.X),
				float32(finish.Y),
				1,
				color.White,
				true,
			)
		}
	}
}
