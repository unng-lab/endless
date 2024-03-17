package endless

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github/unng-lab/madfarmer/internal/astar"
	"github/unng-lab/madfarmer/internal/geom"
)

const (
	UnitStatusUndefined = iota
	UnitStatusRunning
	UnitStatusPaused
	UnitStatusFinished
	UnitStatusIdle
)

type Unit struct {
	Name        string
	Animation   []*ebiten.Image
	PositionX   float64
	PositionY   float64
	SizeX       float64
	SizeY       float64
	DrawOptions ebiten.DrawImageOptions
	Pathing     astar.Astar
	Status      int
}

func (u *Unit) New(positionX float64, positionY float64) Unit {
	var unit Unit
	unit.Name = u.Name
	unit.PositionX = positionX
	unit.PositionY = positionY
	unit.SizeX = u.SizeX
	unit.SizeY = u.SizeY

	unit.Status = UnitStatusRunning

	unit.Pathing = astar.NewAstar(&B)

	err := unit.Pathing.BuildPath(unit.PositionX, unit.PositionY, 40, 20)
	if err != nil {
		panic(err)
	}

	for i := range u.Animation {
		unit.Animation = append(unit.Animation, u.Animation[i])
	}
	return unit
}

func (u *Unit) Draw(screen *ebiten.Image, counter int, camera Camera) bool {
	if u.Status == UnitStatusRunning {
		u.DrawPath(screen, camera)
	}
	if camera.Coordinates.Contains(geom.Pt(u.PositionX, u.PositionY)) {
		return false
	}
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
	return true
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

func (u *Unit) DrawPath(screen *ebiten.Image, camera Camera) {
	if len(u.Pathing.Path) <= 1 {
		panic("path is empty")
	}
	for i := range len(u.Pathing.Path) - 2 {
		if camera.Coordinates.Contains(geom.Pt(u.Pathing.Path[i].X, u.Pathing.Path[i].Y)) ||
			camera.Coordinates.Contains(geom.Pt(u.Pathing.Path[i+1].X, u.Pathing.Path[i+1].Y)) {
			start, finish := camera.GetMiddleInPixels(u.Pathing.Path[i]), camera.GetMiddleInPixels(u.Pathing.Path[i+1])
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
