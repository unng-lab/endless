package endless

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Unit struct {
	Name        string
	Animation   []*ebiten.Image
	PositionX   float64
	PositionY   float64
	DrawOptions ebiten.DrawImageOptions
}

func (u *Unit) New(positionX float64, positionY float64) Unit {
	var unit Unit
	unit.Name = u.Name
	unit.PositionX = positionX
	unit.PositionY = positionY

	for i := range u.Animation {
		unit.Animation = append(unit.Animation, u.Animation[i])
	}
	return unit
}

func (u *Unit) Draw(screen *ebiten.Image, counter int) {
	u.DrawOptions.GeoM.Translate(u.PositionX*C.GetTileSize(), u.PositionY*C.GetTileSize())
	u.DrawOptions.GeoM.Translate(-C.GetPositionX(), -C.GetPositionY())
	u.DrawOptions.GeoM.Scale(
		C.GetScaleFactor(),
		C.GetScaleFactor(),
	)
	screen.DrawImage(u.Animation[counter%len(u.Animation)], &u.DrawOptions)
	u.DrawOptions.GeoM.Reset()
}

func (u *Unit) Update() error {
	//u.DrawOptions.GeoM.Translate(u.PositionX, u.PositionY)
	return nil
}
