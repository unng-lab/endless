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
	ID uint64
	// тип юнита
	Type string
	// Имя юнита
	Name string

	Graphics *Graphics

	Positioning Positioning
	// скорость движения
	Speed float64 // tiles per update tick

	//Ticks экономные тики для отработки игровых событий
	Ticks chan *sync.WaitGroup
	//CameraTicks тики для отработки анимации и тд
	CameraTicks chan struct{}
	Camera      *camera.Camera
	// Карта игры
	Board *board.Board
	// Можно немного пооптимизировать и сделать через глобальную переменную
	Focused bool
	//сколько тиков юнит спит до след изменения
	SleepTicks int

	Tasks TaskList

	RoadTask Road

	MoveChan chan MoveMessage
}

func (u *Unit) Run() {
	go u.run()
}

func (u *Unit) run() {
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
