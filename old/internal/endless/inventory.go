package endless

import (
	"image"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/assets/img"
	"github.com/unng-lab/endless/internal/board"
	"github.com/unng-lab/endless/internal/camera"
	"github.com/unng-lab/endless/internal/unit"
)

const (
	slowness = 1000

	frameOX     = 0
	frameOY     = 32
	frameWidth  = 32
	frameHeight = 32
	frameCount  = 8

	tileMiddleX = 0.5
	tileMiddleY = 0.75
)

var unitSerial atomic.Uint64

func nextSerial() uint64 {
	return unitSerial.Add(1) - 1
}

type Inventory struct {
	Units map[string]Piece
}

func NewInventory(board *board.Board, camera *camera.Camera) *Inventory {
	inventory := &Inventory{
		Units: make(map[string]Piece),
	}

	runner := newRunnerPiece(board, camera)
	inventory.Units[runner.Type()] = runner

	rock := newRockPiece(board, camera)
	inventory.Units[rock.Type()] = rock

	return inventory
}

type Piece interface {
	Type() string
	Unit(
		index int,
		name string,
		cameraTicks chan struct{},
		ticks chan int64,
	) *unit.Unit
}

type pieceBase struct {
	name        string
	board       *board.Board
	camera      *camera.Camera
	graphics    *unit.Graphics
	positioning unit.Positioning
	speed       float64
}

func (p *pieceBase) Type() string {
	return p.name
}

func (p *pieceBase) createUnit(
	index int,
	name string,
	cameraTicks chan struct{},
	ticks chan int64,
) *unit.Unit {
	return &unit.Unit{
		Index:       index,
		ID:          nextSerial(),
		Name:        name,
		Type:        p.name,
		Positioning: p.positioning,
		Speed:       p.speed,
		Graphics:    p.graphics,
		Camera:      p.camera,
		Board:       p.board,
		CameraTicks: cameraTicks,
		Ticks:       ticks,
		Tasks:       unit.NewTaskList(),
	}
}

type Runner struct {
	pieceBase
}

func newRunnerPiece(board *board.Board, camera *camera.Camera) *Runner {
	spriteRunner, err := img.Img("runner.png", 256, 96)
	if err != nil {
		panic(err)
	}

	spriteFocused, err := img.Img("runnerfocused.png", 256, 96)
	if err != nil {
		panic(err)
	}

	graphics := &unit.Graphics{
		Animation:        sliceSpriteFrames(spriteRunner, frameCount),
		FocusedAnimation: sliceSpriteFrames(spriteFocused, frameCount),
	}

	positioning := unit.Positioning{}
	positioning.SizeX = frameWidth / float64(board.TileSize)
	positioning.SizeY = frameHeight / float64(board.TileSize)
	positioning.PositionShiftX = tileMiddleX - positioning.SizeX/2
	positioning.PositionShiftY = tileMiddleY - positioning.SizeY

	return &Runner{
		pieceBase: pieceBase{
			name:        "runner",
			board:       board,
			camera:      camera,
			graphics:    graphics,
			positioning: positioning,
			speed:       1 / float64(ebiten.DefaultTPS) / slowness,
		},
	}
}

func (r *Runner) Unit(
	index int,
	name string,
	cameraTicks chan struct{},
	ticks chan int64,
) *unit.Unit {
	return r.createUnit(index, name, cameraTicks, ticks)
}

type Rock struct {
	pieceBase
}

func newRockPiece(board *board.Board, camera *camera.Camera) *Rock {
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

	graphics := &unit.Graphics{
		Animation:        []*ebiten.Image{frame},
		FocusedAnimation: []*ebiten.Image{frame},
	}

	positioning := unit.Positioning{
		SizeX: 1,
		SizeY: 1,
	}

	return &Rock{
		pieceBase: pieceBase{
			name:        "rock",
			board:       board,
			camera:      camera,
			graphics:    graphics,
			positioning: positioning,
		},
	}
}

func (r *Rock) Unit(
	index int,
	name string,
	cameraTicks chan struct{},
	ticks chan int64,
) *unit.Unit {
	return r.createUnit(index, name, cameraTicks, ticks)
}

func sliceSpriteFrames(sprite *ebiten.Image, count int) []*ebiten.Image {
	frames := make([]*ebiten.Image, 0, count)
	for frameIndex := 0; frameIndex < count; frameIndex++ {
		sx := frameOX + frameIndex*frameWidth
		sy := frameOY
		frame := sprite.SubImage(image.Rect(
			sx,
			sy,
			sx+frameWidth,
			sy+frameHeight,
		)).(*ebiten.Image)
		frames = append(frames, frame)
	}
	return frames
}
