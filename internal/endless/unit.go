package endless

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Unit struct {
	Name        string
	Animation   []*ebiten.Image
	PositionX   float64
	PositionY   float64
	SizeX       float64
	SizeY       float64
	DrawOptions ebiten.DrawImageOptions
}

func (u *Unit) New(positionX float64, positionY float64) Unit {
	var unit Unit
	unit.Name = u.Name
	unit.PositionX = positionX
	unit.PositionY = positionY
	unit.SizeX = u.SizeX
	unit.SizeY = u.SizeY

	for i := range u.Animation {
		unit.Animation = append(unit.Animation, u.Animation[i])
	}
	return unit
}

func (u *Unit) Draw(screen *ebiten.Image, counter int, camera Camera) {
	defer u.DrawOptions.GeoM.Reset()
	u.DrawOptions.GeoM.Scale(
		camera.GetScaleFactor(),
		camera.GetScaleFactor(),
	)
	u.DrawOptions.GeoM.Translate(u.GetDrawPoint(
		camera.GetPositionX(),
		camera.GetPositionY(),
		camera.GetTileSize(),
		camera.GetScaleFactor(),
	))
	screen.DrawImage(u.Animation[counter%len(u.Animation)], &u.DrawOptions)

	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprintf(
			`posX: %0.2f,
	posY: %0.2f,
	TileSize: %0.2f,
	cposX: %0.2f,
	cposY: %0.2f`,
			u.DrawOptions.GeoM.Element(0, 2),
			u.DrawOptions.GeoM.Element(1, 2),
			camera.GetTileSize(),
			camera.GetPositionX(),
			camera.GetPositionY(),
		),
		100,
		0,
	)
}

func (u *Unit) Update() error {
	//u.DrawOptions.GeoM.Translate(u.PositionX, u.PositionY)
	return nil
}

func (u *Unit) GetDrawPoint(
	cameraX, cameraY, tileSize, scale float64,
) (float64, float64) {
	var x, y float64
	x = float64(u.PositionX)*tileSize + tileSize/2 - float64(u.SizeX)*scale/2 - cameraX
	y = float64(u.PositionY)*tileSize + tileSize*3/4 - float64(u.SizeY)*scale - cameraY
	return x, y
}

func (u *Unit) Drawable(cameraX, cameraY, tileSize, scale float64) bool {
	return true
}
