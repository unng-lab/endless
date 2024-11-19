package endless

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/assets/img"
	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/camera"
	"github/unng-lab/madfarmer/internal/unit"
)

const (
	slowness = 10
)

type Inventory struct {
	Units map[string]*unit.Unit
}

func NewInverntory(camera *camera.Camera) *Inventory {
	var i Inventory
	i.Units = make(map[string]*unit.Unit)
	runner := NewRunner(camera)
	i.Units[runner.Name] = runner
	return &i
}

const (
	frameOX     = 0
	frameOY     = 32
	frameWidth  = 32
	frameHeight = 32
	frameCount  = 8
)

func NewRunner(camera *camera.Camera) *unit.Unit {
	var newUnit unit.Unit
	newUnit.Name = "runner"
	newUnit.Camera = camera
	newUnit.SizeX = frameWidth / board.TileSize
	newUnit.SizeY = frameHeight / board.TileSize
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
