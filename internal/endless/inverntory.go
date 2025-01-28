package endless

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/madfarmer/assets/img"
	"github.com/unng-lab/madfarmer/internal/board"
	"github.com/unng-lab/madfarmer/internal/camera"
	"github.com/unng-lab/madfarmer/internal/unit"
)

const (
	slowness = 1
)

type Inventory struct {
	Units map[string]*unit.Unit
}

func NewInverntory(camera *camera.Camera) *Inventory {
	var i Inventory
	i.Units = make(map[string]*unit.Unit)
	runner := NewRunner(camera)
	i.Units[runner.Type] = runner
	rock := NewRock(camera)
	i.Units[rock.Type] = rock
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

func NewRunner(camera *camera.Camera) *unit.Unit {
	var newUnit unit.Unit
	newUnit.Type = "runner"
	newUnit.Camera = camera
	newUnit.Positioning.SizeX = frameWidth / board.TileSize
	newUnit.Positioning.SizeY = frameHeight / board.TileSize
	newUnit.Positioning.PositionShiftX = tileMiddleX - newUnit.Positioning.SizeX/2
	newUnit.Positioning.PositionShiftY = tileMiddleY - newUnit.Positioning.SizeY
	newUnit.Speed = 1 / float64(ebiten.DefaultTPS) / slowness
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

	newUnit.Graphics = &graphics

	return &newUnit
}

func NewRock(camera *camera.Camera) *unit.Unit {
	var newUnit unit.Unit
	newUnit.Type = "rock"
	newUnit.Camera = camera
	newUnit.Positioning.SizeX, newUnit.Positioning.SizeY = 1, 1
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
	newUnit.Graphics = &graphics

	return &newUnit
}
