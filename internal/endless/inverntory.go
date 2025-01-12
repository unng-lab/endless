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

	tileMiddleX = 0
	tileMiddleY = 0.25
)

func NewRunner(camera *camera.Camera) *unit.Unit {
	var newUnit unit.Unit
	newUnit.Type = "runner"
	newUnit.Camera = camera
	newUnit.SizeX = frameWidth / board.TileSize
	newUnit.SizeY = frameHeight / board.TileSize
	newUnit.PositionShiftX = tileMiddleX - newUnit.SizeX/2
	newUnit.PositionShiftY = tileMiddleY - newUnit.SizeY
	newUnit.Speed = 1 / float64(ebiten.DefaultTPS) / slowness
	spriteRunner, err := img.Img("runner.png", 256, 96)
	if err != nil {
		panic(err)
	}
	spriteFocused, err := img.Img("runnerfocused.png", 256, 96)
	if err != nil {
		panic(err)
	}
	for i := range frameCount {
		sx, sy := frameOX+i*frameWidth, frameOY
		frame := spriteRunner.SubImage(image.Rect(
			sx,
			sy,
			sx+frameWidth,
			sy+frameHeight,
		)).(*ebiten.Image)
		newUnit.Animation = append(newUnit.Animation, frame)

		frameFocused := spriteFocused.SubImage(image.Rect(
			sx,
			sy,
			sx+frameWidth,
			sy+frameHeight,
		)).(*ebiten.Image)
		newUnit.FocusedAnimation = append(newUnit.FocusedAnimation, frameFocused)
	}

	return &newUnit
}

func NewRock(camera *camera.Camera) *unit.Unit {
	var newUnit unit.Unit
	newUnit.Type = "rock"
	newUnit.Camera = camera
	newUnit.SizeX, newUnit.SizeY = 1, 1
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

	newUnit.Animation = append(newUnit.Animation, frame)
	newUnit.FocusedAnimation = append(newUnit.FocusedAnimation, frame)

	return &newUnit
}
