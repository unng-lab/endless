package endless

import (
	"image"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/madfarmer/assets/img"
	"github.com/unng-lab/madfarmer/internal/board"
	"github.com/unng-lab/madfarmer/internal/camera"
	"github.com/unng-lab/madfarmer/internal/unit"
)

const (
	slowness = 1000
)

var serial atomic.Uint64

func inc() uint64 {
	return serial.Add(1) - 1
}

type Inventory struct {
	Units map[string]Piece
}

func NewInverntory(board *board.Board, camera *camera.Camera) *Inventory {
	var i Inventory
	i.Units = make(map[string]Piece)
	runner := NewRunner(board, camera)
	i.Units[runner.Type()] = runner
	rock := NewRock(board, camera)
	i.Units[rock.Type()] = rock
	return &i
}

const (
	frameOX     = 0
	frameOY     = 32
	frameWidth  = 32
	frameHeight = 32
	frameCount  = 8

	tileMiddleX = 0.5
	tileMiddleY = 0.75
)

type Piece interface {
	Type() string
	Unit(
		index int,
		name string,
		moveChan chan unit.MoveMessage,
		cameraTicks chan struct{},
		ticks chan int64,
	) *unit.Unit
}

type Runner struct {
	Board       *board.Board
	Camera      *camera.Camera
	Graphics    *unit.Graphics
	Positioning unit.Positioning
	Speed       float64
	Name        string
}

func (r *Runner) Type() string {
	return r.Name
}

func NewRunner(board *board.Board, camera *camera.Camera) *Runner {
	var r Runner
	r.Board = board
	r.Camera = camera
	r.Name = "runner"
	spriteRunner, err := img.Img("runner.png", 256, 96)
	if err != nil {
		panic(err)
	}
	spriteFocused, err := img.Img("runnerfocused.png", 256, 96)
	if err != nil {
		panic(err)
	}
	graphics := unit.Graphics{
		Animation:        make([]*ebiten.Image, 0, frameCount),
		FocusedAnimation: make([]*ebiten.Image, 0, frameCount),
	}

	for i := range frameCount {
		sx, sy := frameOX+i*frameWidth, frameOY
		frame := spriteRunner.SubImage(image.Rect(
			sx,
			sy,
			sx+frameWidth,
			sy+frameHeight,
		)).(*ebiten.Image)
		graphics.Animation = append(graphics.Animation, frame)

		frameFocused := spriteFocused.SubImage(image.Rect(
			sx,
			sy,
			sx+frameWidth,
			sy+frameHeight,
		)).(*ebiten.Image)
		graphics.FocusedAnimation = append(graphics.FocusedAnimation, frameFocused)
	}

	r.Graphics = &graphics

	r.Positioning.SizeX = frameWidth / float64(board.TileSize)
	r.Positioning.SizeY = frameHeight / float64(board.TileSize)
	r.Positioning.PositionShiftX = tileMiddleX - r.Positioning.SizeX/2
	r.Positioning.PositionShiftY = tileMiddleY - r.Positioning.SizeY
	r.Speed = 1 / float64(ebiten.DefaultTPS) / slowness

	return &r
}

func (r *Runner) Unit(
	index int,
	name string,
	moveChan chan unit.MoveMessage,
	cameraTicks chan struct{},
	ticks chan int64,
) *unit.Unit {
	var newUnit unit.Unit
	newUnit.Index = index
	newUnit.ID = inc()
	newUnit.Name = name
	newUnit.Type = r.Type()
	newUnit.Positioning = r.Positioning
	newUnit.Speed = r.Speed
	newUnit.Graphics = r.Graphics
	newUnit.Camera = r.Camera
	newUnit.Board = r.Board
	newUnit.MoveChan = moveChan
	newUnit.CameraTicks = cameraTicks
	newUnit.Ticks = ticks
	newUnit.Tasks = unit.NewTaskList()
	return &newUnit
}

type Rock struct {
	Board       *board.Board
	Camera      *camera.Camera
	Graphics    *unit.Graphics
	Positioning unit.Positioning
	Speed       float64
	Name        string
}

func (r Rock) Type() string {
	return r.Name
}

func (r Rock) Unit(
	index int,
	name string,
	moveChan chan unit.MoveMessage,
	cameraTicks chan struct{},
	ticks chan int64,
) *unit.Unit {
	var newUnit unit.Unit
	newUnit.Index = index
	newUnit.ID = inc()
	newUnit.Name = name
	newUnit.Type = r.Type()
	newUnit.Positioning = r.Positioning
	newUnit.Speed = r.Speed
	newUnit.Graphics = r.Graphics
	newUnit.Camera = r.Camera
	newUnit.Board = r.Board
	newUnit.MoveChan = moveChan
	newUnit.CameraTicks = cameraTicks
	newUnit.Ticks = ticks
	newUnit.Tasks = unit.NewTaskList()
	return &newUnit
}

func NewRock(board *board.Board, camera *camera.Camera) *Rock {
	var r Rock
	r.Name = "rock"
	r.Board = board
	r.Camera = camera
	r.Positioning.SizeX, r.Positioning.SizeY = 1, 1
	spriteRocks, err := img.Img("rocks.png", 32, 32)
	if err != nil {
		panic(err)
	}

	frame := spriteRocks.SubImage(image.Rect(
		0,
		0,
		16,
		16,
	)).(*ebiten.Image)

	graphics := unit.Graphics{
		Animation:        make([]*ebiten.Image, 0, frameCount),
		FocusedAnimation: make([]*ebiten.Image, 0, frameCount),
	}
	graphics.Animation = append(graphics.Animation, frame)
	graphics.FocusedAnimation = append(graphics.FocusedAnimation, frame)
	r.Graphics = &graphics

	return &r
}
