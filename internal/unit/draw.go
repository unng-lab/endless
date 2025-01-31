package unit

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"

	"github.com/unng-lab/madfarmer/internal/geom"
)

func (u *Unit) Draw(screen *ebiten.Image, counter int) bool {
	drawOptions := drawOptionsPool.Get().(*ebiten.DrawImageOptions)
	defer func() {
		drawOptions.GeoM.Reset()
		drawOptionsPool.Put(drawOptions)
	}()

	drawOptions.GeoM.Scale(
		u.Camera.ScaleFactor(),
		u.Camera.ScaleFactor(),
	)
	drawPoint := u.GetDrawPoint()
	drawOptions.GeoM.Translate(drawPoint.X, drawPoint.Y)
	anim := counter & 0xFFFF >> 6
	if u.Focused {
		screen.DrawImage(u.Graphics.FocusedAnimation[anim%len(u.Graphics.FocusedAnimation)], drawOptions)
		u.DrawTitle(screen, drawOptions)
	} else {
		screen.DrawImage(u.Graphics.Animation[anim%len(u.Graphics.Animation)], drawOptions)
	}

	if drawRect {
		u.drawRect(screen)
	}
	return true
}

func (u *Unit) DrawTitle(screen *ebiten.Image, drawOptions *ebiten.DrawImageOptions) {
	// Создаем шрифт
	fontFace := basicfont.Face7x13

	// Рисуем текст на экране
	text.DrawWithOptions(screen, u.Name, fontFace, drawOptions)
}

func (u *Unit) drawRect(screen *ebiten.Image) {
	leftAngle := u.GetDrawPoint()
	posX, posY := leftAngle.X, leftAngle.Y

	vector.StrokeLine(
		screen,
		float32(posX),
		float32(posY),
		float32(posX+u.Camera.TileSize()*u.Positioning.SizeX),
		float32(posY),
		1,
		color.White,
		false,
	)
	vector.StrokeLine(
		screen,
		float32(posX+u.Camera.TileSize()*u.Positioning.SizeX),
		float32(posY),
		float32(posX+u.Camera.TileSize()*u.Positioning.SizeX),
		float32(posY+u.Camera.TileSize()*u.Positioning.SizeY),
		1,
		color.White,
		false,
	)
	vector.StrokeLine(
		screen,
		float32(posX),
		float32(posY),
		float32(posX),
		float32(posY+u.Camera.TileSize()*u.Positioning.SizeY),
		1,
		color.White,
		false,
	)
	vector.StrokeLine(
		screen,
		float32(posX),
		float32(posY+u.Camera.TileSize()*u.Positioning.SizeY),
		float32(posX+u.Camera.TileSize()*u.Positioning.SizeX),
		float32(posY+u.Camera.TileSize()*u.Positioning.SizeY),
		1,
		color.White,
		false,
	)
}

func (u *Unit) GetDrawPoint() geom.Point {
	drawPoint := u.Camera.PointToCameraPixel(geom.Point{
		X: u.Positioning.Position.X + u.Positioning.PositionShiftX + u.Positioning.PositionShiftModX,
		Y: u.Positioning.Position.Y + u.Positioning.PositionShiftY + u.Positioning.PositionShiftModY,
	})
	return drawPoint
}
