package unit

import (
	"sync"
	"sync/atomic"

	"github.com/brianvoe/gofakeit/v7"
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
	ID int
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
	// Статус что находится на экране игры
	OnBoard atomic.Bool
	// Можно немного пооптимизировать и сделать через глобальную переменную
	Focused bool

	//сколько тиков юнит спит до след изменения
	SleepTicks int

	Tasks TaskList

	RoadTask Road

	MoveChan chan MoveMessage
}

func (u *Unit) New(
	id int,
	position geom.Point,
	b *board.Board,
	moveChan chan MoveMessage,
) *Unit {
	var unit Unit
	unit.ID = id
	unit.Type = u.Type
	unit.Name = gofakeit.Name()
	unit.Camera = u.Camera
	unit.Board = b
	unit.MoveChan = moveChan
	unit.CameraTicks = make(chan struct{}, 1)
	unit.Ticks = make(chan *sync.WaitGroup, 1)

	unit.Speed = u.Speed

	unit.Graphics = u.Graphics

	unit.Positioning = u.Positioning

	unit.Tasks = NewTaskList()

	unit.SetTask()

	return &unit
}

func (u *Unit) SetOnBoard(b bool) {
	u.OnBoard.Store(b)
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
