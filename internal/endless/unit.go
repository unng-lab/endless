package endless

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github/unng-lab/madfarmer/internal/astar"
	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/camera"
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
	Name            string
	Animation       []*ebiten.Image
	PositionX       float64
	PositionY       float64
	SizeX           float64
	SizeY           float64
	Speed           float64 // tiles per update tick
	PositionShiftX  float64 // in tiles
	PositionShiftY  float64 // in tiles
	CurrentPosition geom.Point
	DrawOptions     ebiten.DrawImageOptions
	Pathing         astar.Astar
	Status          int
}

func (u *Unit) New(positionX float64, positionY float64, tileSize float64) Unit {
	var unit Unit
	unit.Name = u.Name
	unit.PositionX = positionX
	unit.PositionY = positionY
	unit.SizeX = u.SizeX
	unit.SizeY = u.SizeY
	unit.Speed = 1 / ebiten.DefaultTPS
	unit.Pathing = astar.NewAstar(&board.B)
	if unit.PositionX == 2078 {
		unit.Status = UnitStatusRunning
		err := unit.Pathing.BuildPath(
			unit.PositionX,
			unit.PositionY,
			board.CountTile/2+40,
			board.CountTile/2+20,
		)
		if err != nil {
			panic(err)
		}
	} else {
		unit.Status = UnitStatusRunning
		err := unit.Pathing.BuildPath(
			unit.PositionX,
			unit.PositionY,
			float64(rand.Intn(board.CountTile)),
			float64(rand.Intn(board.CountTile)),
		)
		if err != nil {
			panic(err)
		}
	}

	for i := range u.Animation {
		unit.Animation = append(unit.Animation, u.Animation[i])
	}
	unit.PositionShiftX = -u.SizeX / 2
	unit.PositionShiftY = tileSize/4 - u.SizeY

	return unit
}

func (u *Unit) Draw(screen *ebiten.Image, counter int, camera camera.Camera) bool {
	if u.Status == UnitStatusRunning {
		u.DrawPath(screen, camera)
	}
	if !camera.Coordinates.Contains(geom.Pt(u.PositionX, u.PositionY)) {
		return false
	}
	defer u.DrawOptions.GeoM.Reset()
	u.DrawOptions.GeoM.Scale(
		camera.GetScaleFactor(),
		camera.GetScaleFactor(),
	)
	drawPoint := u.GetDrawPoint(camera)
	u.DrawOptions.GeoM.Translate(drawPoint.X, drawPoint.Y)
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
	camera camera.Camera,
) geom.Point {
	drawPoint := camera.GetMiddleInPixels(geom.Pt(u.PositionX, u.PositionY))
	drawPoint.X = drawPoint.X + u.PositionShiftX*camera.GetScaleFactor()
	drawPoint.Y = drawPoint.Y + u.PositionShiftY*camera.GetScaleFactor()
	return drawPoint
}

func (u *Unit) Drawable(cameraX, cameraY, tileSize, scale float64) bool {
	return true
}

func (u *Unit) DrawPath(screen *ebiten.Image, camera camera.Camera) {
	if len(u.Pathing.Path) <= 1 {
		panic("path is empty")
	}
	start := camera.GetMiddleInPixels(geom.Pt(u.PositionX, u.PositionY))
	for i := range len(u.Pathing.Path) - 2 {
		finish := camera.GetMiddleInPixels(u.Pathing.Path[i+1])
		if camera.Pixels.Contains(start) ||
			camera.Pixels.Contains(finish) {
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
		start = finish
	}
}
