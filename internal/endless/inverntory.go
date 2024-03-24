package endless

import (
	"bytes"
	"image"
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/camera"
	"github/unng-lab/madfarmer/internal/unit"
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
	img, _, err := image.Decode(bytes.NewReader(images.Runner_png))
	if err != nil {
		slog.Error("image.Decode", err)
	}
	sprite := ebiten.NewImageFromImage(img)
	for i := range frameCount {
		sx, sy := frameOX+i*frameWidth, frameOY
		frame := sprite.SubImage(image.Rect(
			sx,
			sy,
			sx+frameWidth,
			sy+frameHeight,
		)).(*ebiten.Image)
		newUnit.Animation = append(newUnit.Animation, frame)
	}

	return &newUnit
}
