package runner

import (
	"bytes"
	"image"
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

// var _ game.Unit = (*Default)(nil)
//var _ scr.Unit = (*Default)(nil)

const id = "runner"

const (
	frameOX     = 0
	frameOY     = 32
	frameWidth  = 32
	frameHeight = 32
	frameCount  = 8
)

type Default struct {
	log      *slog.Logger
	cfg      Config
	sprite   *ebiten.Image
	position image.Point
}

func (d *Default) Position() image.Point {
	return image.Pt(2000, 2000)
}

type Config interface {
}

func (d *Default) ID() string {
	return id
}

func New(log *slog.Logger, cfg Config) *Default {
	var d Default
	d.log = log.With("runner", "Default")
	d.cfg = cfg

	img, _, err := image.Decode(bytes.NewReader(images.Runner_png))
	if err != nil {
		d.log.Error("image.Decode", "err", err)
	}
	d.sprite = ebiten.NewImageFromImage(img)

	return &d
}

func (d *Default) Draw(
	screen *ebiten.Image,
	counter int64,
) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(frameWidth)/2, -float64(frameHeight)/2)
	//op.GeoM.Translate(float64(d.Position().X+div.X), float64(d.Position().Y+div.Y))
	op.GeoM.Translate(float64(d.Position().X), float64(d.Position().Y))
	op.GeoM.Scale(float64(10), float64(10))
	i := (counter / 5) % frameCount
	sx, sy := frameOX+i*frameWidth, frameOY
	screen.DrawImage(d.sprite.SubImage(image.Rect(int(sx), sy, int(sx+frameWidth), sy+frameHeight)).(*ebiten.Image), op)
}
