package endless

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Unit struct {
	Name        string
	Animation   []*ebiten.Image
	PositionX   int
	PositionY   int
	SizeX       int
	SizeY       int
	DrawOptions ebiten.DrawImageOptions
}

func (u *Unit) New(positionX int, positionY int) Unit {
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
	posX := float64(u.PositionX)*camera.GetTileSize() - camera.GetTileSize()/2 - float64(u.SizeX)/2 - camera.GetPositionX()
	posY := float64(u.PositionY)*camera.GetTileSize() - camera.GetTileSize()/4 - float64(u.SizeY) - camera.GetPositionY()

	u.DrawOptions.GeoM.Translate(float64(posX), float64(posY))
	//u.DrawOptions.GeoM.Scale(
	//	camera.GetScaleFactor(),
	//	camera.GetScaleFactor(),
	//)
	screen.DrawImage(u.Animation[counter%len(u.Animation)], &u.DrawOptions)
	u.DrawOptions.GeoM.Reset()

	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprintf(
			`posX: %0.2f, 
posY: %0.2f,
TileSize: %0.2f,
cposX: %0.2f, 
cposY: %0.2f`,
			posX,
			posY,
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
