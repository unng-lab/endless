package endless

import (
	"github.com/hajimehoshi/ebiten/v2"
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
	u.DrawOptions.GeoM.Translate((u.PositionX-0.5)*camera.GetTileSize()-u.SizeX/2,
		(u.PositionY-0.25)*camera.GetTileSize()-u.SizeY)
	u.DrawOptions.GeoM.Translate(-camera.GetPositionX(), -camera.GetPositionY())
	//u.DrawOptions.GeoM.Scale(
	//	camera.GetScaleFactor(),
	//	camera.GetScaleFactor(),
	//)
	screen.DrawImage(u.Animation[counter%len(u.Animation)], &u.DrawOptions)
	u.DrawOptions.GeoM.Reset()
}

func (u *Unit) Update() error {
	//u.DrawOptions.GeoM.Translate(u.PositionX, u.PositionY)
	return nil
}
