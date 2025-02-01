package unit

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/madfarmer/internal/board"
	"github.com/unng-lab/madfarmer/internal/camera"
	"github.com/unng-lab/madfarmer/internal/geom"
)

const (
	drawRect = true
)

var drawOptionsPool = sync.Pool{
	New: func() any {
		return &ebiten.DrawImageOptions{}
	},
}

type Graphics struct {
	Animation        []*ebiten.Image
	FocusedAnimation []*ebiten.Image
}
type Positioning struct {
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
}

type Unit struct {
	ID    uint64
	Index int
	// тип юнита
	Type string
	// Имя юнита
	Name string

	Graphics *Graphics

	Positioning Positioning
	// скорость движения
	Speed float64 // tiles per update tick

	//Ticks экономные тики для отработки игровых событий
	Ticks chan int64
	//CameraTicks тики для отработки анимации и тд
	CameraTicks chan struct{}
	// WG завершение рабочего цикла
	WG *sync.WaitGroup

	Camera *camera.Camera
	// Карта игры
	Board *board.Board
	// Можно немного пооптимизировать и сделать через глобальную переменную
	Focused bool
	//сколько тиков юнит спит до след изменения
	SleepTicks int

	Tasks TaskList

	RoadTask Road

	//MoveChan chan MoveMessage
}

func (u *Unit) Run() {
	go u.run()
}

func (u *Unit) run() {
	for {
		select {
		case <-u.Ticks:
			func() {
				defer u.WG.Done()
				n, err := u.Update()
				if err != nil {
					return
				}
				u.SleepTicks = n
			}()

		case <-u.CameraTicks:
			u.OnBoardUpdate()
		}
	}
}

func (u *Unit) Play(gameTick int64) {
	defer u.WG.Done()
	n, err := u.Update()
	if err != nil {
		return
	}
	u.SleepTicks = n
}

func (u *Unit) OnBoardUpdate() {
	var curTask Task
	u.Focused = u.isFocused(u.Camera.Cursor)

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
	Max := Min.Add(geom.Pt(u.Positioning.SizeX*u.Camera.TileSize(), u.Positioning.SizeY*u.Camera.TileSize()))
	return geom.Rectangle{Min: Min, Max: Max}
}

func (u *Unit) isFocused(p geom.Point) bool {
	return p.In(u.Rect())
}

type MoveMessage struct {
	U        *Unit
	From, To geom.Point
}

func (u *Unit) Cost() float64 {
	return math.Inf(1)
}

type Status struct {
	OnBoard  bool
	Position geom.Point
}

func (u *Unit) Process() Status {
	var ustatus Status
	ustatus.OnBoard = u.checkOnBoard()
	ustatus.Position = u.Positioning.Position
	return ustatus
}

func (u *Unit) checkOnBoard() bool {
	return u.Positioning.Position.In(geom.Rectangle{
		Min: u.Camera.Coordinates.Min,
		Max: u.Camera.Coordinates.Max,
	})
}
