package unit

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/internal/astar"
	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/camera"
	"github/unng-lab/madfarmer/internal/ch"
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
	ID   int
	Name string
	// обычная анимация
	Animation []*ebiten.Image
	// анимация с фокусом
	FocusedAnimation []*ebiten.Image
	// Целочисленные координаты
	Position geom.Point
	// Размеры иконки
	SizeX float64
	SizeY float64
	// сдвиг иконки относительно позиции
	PositionShiftX float64 // in tiles
	PositionShiftY float64 // in tiles
	// сдвиг иконки относительно задания
	PositionShiftModX float64 // in tiles
	PositionShiftModY float64 // in tiles
	// скорость движения
	Speed float64 // tiles per update tick

	DrawOptions ebiten.DrawImageOptions
	//deprecated
	Pathing astar.Astar
	//deprecated
	Status int
	//Ticks экономные тики для отработки игровых событий
	Ticks chan *sync.WaitGroup
	//CameraTicks тики для отработки анимации и тд
	CameraTicks chan struct{}
	Camera      *camera.Camera
	OnBoard     atomic.Bool
	// Можно немного пооптимизировать и сделать через глобальную переменную
	Focused bool

	//Analitics
	AnaliticsDB *ch.AnaliticsDB

	//deprecated
	CurTarget geom.Point

	//сколько тиков юнит спит до след изменения
	SleepTicks int

	Tasks TaskList

	RoadTask Road

	MoveChan chan MoveMessage
}

func (u *Unit) New(
	id int,
	positionX float64,
	positionY float64,
	b *board.Board,
	moveChan chan MoveMessage,
// db *ch.AnaliticsDB,
) *Unit {
	var unit Unit
	unit.ID = id
	unit.Name = gofakeit.Name()
	unit.Camera = u.Camera
	unit.MoveChan = moveChan
	unit.CameraTicks = make(chan struct{}, 1)
	unit.Ticks = make(chan *sync.WaitGroup, 1)

	unit.SizeX = u.SizeX
	unit.SizeY = u.SizeY
	unit.Speed = u.Speed

	for i := range u.Animation {
		unit.Animation = append(unit.Animation, u.Animation[i])
	}

	for i := range u.FocusedAnimation {
		unit.FocusedAnimation = append(unit.FocusedAnimation, u.FocusedAnimation[i])
	}

	unit.PositionShiftX = 0.5 - u.SizeX/2
	unit.PositionShiftY = 0.75 - u.SizeY
	//unit.AnaliticsDB = db

	unit.Tasks = NewTaskList()

	unit.Relocate(geom.Pt(0, 0), geom.Pt(positionX, positionY))

	// временное для добавление сходу задания на попиздовать куда то
	unit.RoadTask = NewRoad(b, &unit)
	if err := unit.RoadTask.Path(
		geom.Pt(
			float64(rand.Intn(board.CountTile)),
			float64(rand.Intn(board.CountTile)),
		)); err != nil {
		panic(err)
	}

	unit.Tasks.Add(&unit.RoadTask)

	return &unit
}

func (u *Unit) SetOnBoard(b bool) {
	u.OnBoard.Store(b)
}

// deprecated
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
	//u.AnaliticsDB.AddPath(&ch.Path{
	//	UnitID: u.ID,
	//	X:      u.Position.X,
	//	Y:      u.Position.Y,
	//	Cost:   heuristic(u.Position, u.CurTarget),
	//	GoalX:  u.CurTarget.X,
	//	GoalY:  u.CurTarget.Y,
	//})
}

func (u *Unit) Run(wg chan *sync.WaitGroup) {
	go u.run(wg)
}

func (u *Unit) run(wg chan *sync.WaitGroup) {
	u.Ticks = wg
	for {
		select {
		case tick := <-u.Ticks:
			n, err := u.Update()
			if err != nil {
				return
			}
			u.SleepTicks = n
			tick.Done()
		case <-u.CameraTicks:
			u.OnBoardUpdate()
		}
	}
}

func (u *Unit) OnBoardUpdate() {
	var curTask Task

	u.Focused = false
	if u.isFocused(u.Camera.Cursor) {
		u.Focused = true
	}

	if curTask = u.Tasks.Current(); curTask == nil {
		return
	}

	err := curTask.Update(u)
	if err != nil {

		return
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

func heuristic(current geom.Point, goal geom.Point) float64 {
	return math.Sqrt((current.X-goal.X)*(current.X-goal.X) +
		(current.Y-goal.Y)*(current.Y-goal.Y))

}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

type MoveMessage struct {
	U        *Unit
	From, To geom.Point
}
