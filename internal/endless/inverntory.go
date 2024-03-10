package endless

import (
	"bytes"
	"image"
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

var I Inverntory

type Inverntory struct {
	Units map[string]*Unit
}

func NewInverntory() {
	I.Units = make(map[string]*Unit)
	runner := NewRunner()
	I.Units[runner.Name] = runner
}

const (
	frameOX     = 0
	frameOY     = 32
	frameWidth  = 32
	frameHeight = 32
	frameCount  = 8
)

func NewRunner() *Unit {
	var u Unit
	u.Name = "runner"
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
		u.Animation = append(u.Animation, frame)
	}

	return &u
}
