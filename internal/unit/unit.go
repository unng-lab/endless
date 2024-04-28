package unit

import (
	"image/color"
	"math/rand"
	"sync"

	"github.com/brianvoe/gofakeit/v7"
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

const (
	drawRect = true
)

type Unit struct {
	ID               int
	Name             string
	Animation        []*ebiten.Image
	FocusedAnimation []*ebiten.Image
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
	Ticks            chan *WG
	Camera           *camera.Camera
	OnBoard          bool
	// Можно немного пооптимизировать и сделать через глобальную переменную
	Focused bool
}

type WG struct {
	WG *sync.WaitGroup
	S  int
}

func (wg *WG) Done(sleeper int) {
	wg.S = sleeper
	wg.WG.Done()
}

func (u *Unit) New(id int, positionX float64, positionY float64, b *board.Board) Unit {
	var unit Unit
	unit.ID = id
	unit.Name = gofakeit.Name()
	unit.Camera = u.Camera
	//unit.Ticks = wg
	unit.Position.X = positionX
	unit.Position.Y = positionY
	unit.SizeX = u.SizeX
	unit.SizeY = u.SizeY
	unit.Speed = 1 / float64(ebiten.DefaultTPS)
	unit.Pathing = astar.NewAstar(b)
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

	for i := range u.FocusedAnimation {
		unit.FocusedAnimation = append(unit.FocusedAnimation, u.FocusedAnimation[i])
	}

	unit.PositionShiftX = 0.5 - u.SizeX/2
	unit.PositionShiftY = 0.75 - u.SizeY

	return unit
}

func (u *Unit) Draw(screen *ebiten.Image, counter int) bool {
	if u.Status == UnitStatusRunning {
		//u.DrawPath(screen, camera)
	}

	defer u.DrawOptions.GeoM.Reset()
	u.DrawOptions.GeoM.Scale(
		u.Camera.ScaleFactor(),
		u.Camera.ScaleFactor(),
	)
	drawPoint := u.GetDrawPoint()
	u.DrawOptions.GeoM.Translate(drawPoint.X, drawPoint.Y)
	if u.Focused {
		screen.DrawImage(u.FocusedAnimation[counter%len(u.FocusedAnimation)], &u.DrawOptions)
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
func (u *Unit) Run(wg chan *WG) {
	go u.run(wg)
}

func (u *Unit) run(wg chan *WG) {
	u.Ticks = wg
	for {
		select {
		case tick := <-u.Ticks:
			err := u.Update()
			if err != nil {
				return
			}
			tick.Done(0)
		}
	}
}

func (u *Unit) Rect() geom.Rectangle {
	Min := u.GetDrawPoint()
	Max := Min.Add(geom.Pt(u.SizeX*u.Camera.TileSize(), u.SizeY*u.Camera.TileSize()))
	return geom.Rectangle{Min: Min, Max: Max}
}

func (u *Unit) isFocused(p geom.Point) bool {
	if p.In(u.Rect()) {
		return true
	}
	return false
}
