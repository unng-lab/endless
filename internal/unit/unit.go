package unit

import (
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
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
	ID               int
	Name             string
	Animation        []*ebiten.Image
	Position         geom.Point
	SizeX            float64
	SizeY            float64
	Speed            float64 // tiles per update tick
	PositionShiftX   float64 // in tiles
	PositionShiftY   float64 // in tiles
	AbsolutePosition geom.Point
	DrawOptions      ebiten.DrawImageOptions
	Pathing          astar.Astar
	Status           int
}

func (u *Unit) New(id int, positionX float64, positionY float64) Unit {
	var unit Unit
	unit.ID = id
	unit.Name = u.Name
	unit.Position.X = positionX
	unit.Position.Y = positionY
	unit.SizeX = u.SizeX
	unit.SizeY = u.SizeY
	unit.Speed = 10 / float64(ebiten.DefaultTPS)
	unit.Pathing = astar.NewAstar(&board.B)
	if unit.Position.X == 2058 {
		unit.Status = UnitStatusRunning
		err := unit.Pathing.BuildPath(
			unit.Position.X,
			unit.Position.Y,
			board.CountTile/2+40,
			board.CountTile/2+20,
		)
		if err != nil {
			panic(err)
		}
	} else {
		unit.Status = UnitStatusRunning
		err := unit.Pathing.BuildPath(
			unit.Position.X,
			unit.Position.Y,
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
	unit.PositionShiftX = 0.5 - u.SizeX/2
	unit.PositionShiftY = 0.75 - u.SizeY

	return unit
}

func (u *Unit) Draw(screen *ebiten.Image, counter int, camera camera.Camera) bool {
	if u.Status == UnitStatusRunning {
		//u.DrawPath(screen, camera)
	}
	if camera.Coordinates.ContainsOR(geom.Pt(u.Position.X, u.Position.Y)) {
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

	//ebitenutil.DebugPrintAt(
	//	screen,
	//	fmt.Sprintf(
	//		`posX: %0.2f,
	//posY: %0.2f,
	//TileSize: %0.2f,
	//cposX: %0.2f,
	//cposY: %0.2f`,
	//		u.DrawOptions.GeoM.Element(0, 2),
	//		u.DrawOptions.GeoM.Element(1, 2),
	//		camera.GetTileSize(),
	//		camera.GetPositionX(),
	//		camera.GetPositionY(),
	//	),
	//	100,
	//	0,
	//)
	return true
}

func (u *Unit) Update() error {
	switch u.Status {
	case UnitStatusRunning:
		u.Move()
	case UnitStatusIdle:
		err := u.Pathing.BuildPath(u.Position.X, u.Position.Y, float64(rand.Intn(board.CountTile)), float64(rand.Intn(board.CountTile)))
		if err != nil {
			return err
		}
		u.Status = UnitStatusRunning
	}

	return nil
}

func (u *Unit) GetDrawPoint(
	camera camera.Camera,
) geom.Point {
	drawPoint := camera.PointToCameraPixel(geom.Point{
		X: u.Position.X + u.PositionShiftX,
		Y: u.Position.Y + u.PositionShiftY,
	})
	return drawPoint
}

func (u *Unit) Drawable(cameraX, cameraY, tileSize, scale float64) bool {
	return true
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

func (u *Unit) Move() {
	distance := u.Speed * u.Pathing.B.Cells[int(u.Position.X)][int(u.Position.Y)].MoveCost()
	part := distance / u.Position.Length(u.Pathing.Path[len(u.Pathing.Path)-2])
	if part > 1 {
		if len(u.Pathing.Path) > 2 {
			u.Pathing.Path = u.Pathing.Path[:len(u.Pathing.Path)-1]
			u.Move()
		} else {
			u.Status = UnitStatusIdle
			u.Position.X = u.Pathing.Path[len(u.Pathing.Path)-2].X
			u.Position.Y = u.Pathing.Path[len(u.Pathing.Path)-2].Y
		}
	} else if part == 1 {
		u.Pathing.Path = u.Pathing.Path[:len(u.Pathing.Path)-1]
		u.Position.X = u.Pathing.Path[len(u.Pathing.Path)-2].X
		u.Position.Y = u.Pathing.Path[len(u.Pathing.Path)-2].Y
	} else {
		u.Position.X = u.Position.X + part*(u.Pathing.Path[len(u.Pathing.Path)-2].X-u.Position.X)
		u.Position.Y = u.Position.Y + part*(u.Pathing.Path[len(u.Pathing.Path)-2].Y-u.Position.Y)
	}
}
