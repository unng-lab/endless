package scr

import (
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github/unng-lab/madfarmer/assets/img"
	"github/unng-lab/madfarmer/internal/ebitenfx"
	"github/unng-lab/madfarmer/internal/units/runner"
	"github/unng-lab/madfarmer/internal/window"
)

var _ ebitenfx.Screen = (*Canvas)(nil)

type Config interface {
}
type Canvas struct {
	log     *slog.Logger
	cfg     Config
	window  *window.Default
	camera  Camera
	bgImage *ebiten.Image
	counter int64
	units   []Unit
}

type Unit interface {
	Position() image.Point
	Draw(*ebiten.Image, int64)
}

func (c *Canvas) Draw(screen *ebiten.Image) {
	repeat := 3
	w, h := c.bgImage.Bounds().Dx(), c.bgImage.Bounds().Dy()
	dX := int(c.camera.position[1]) / w
	dY := int(c.camera.position[0]) / h
	for j := repeat - 5 + dX; j < repeat+dX; j++ {
		for i := repeat + dY - 5; i < repeat+dY; i++ {
			if j == 1 && i == 1 {
				for _, u := range c.units {
					u.Draw(c.bgImage, c.counter)
				}
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(w*i), float64(h*j))
			op.GeoM.Translate(-c.camera.position[0], -c.camera.position[1])
			op.GeoM.Scale(
				math.Pow(1.01, float64(c.camera.zoomFactor)),
				math.Pow(1.01, float64(c.camera.zoomFactor)),
			)
			//op.GeoM.Translate(offsetX, offsetY)
			screen.DrawImage(c.bgImage, op)
		}
	}

	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprintf(
			"T: %.1f, R: %d, S: %d,dX: %d,S: %d",
			c.camera.position, c.camera.rotation, c.camera.zoomFactor,
		),
		0,
		100,
	)

	//options := &ebiten.DrawImageOptions{
	//	GeoM: c.camera.worldMatrix(),
	//}
	//screen.DrawImage(uiImage.SubImage(image.Rect(0, 0, int(c.window.Width.Load()), int(c.window.Height.Load()))).(*ebiten.Image), options)
	//screen.DrawImage(c.bgImage, options)
}

type Game interface {
	Run() error
}

func New(log *slog.Logger, cfg Config, window *window.Default, runner *runner.Default) *Canvas {
	var c Canvas
	c.log = log.With("scr", "Canvas")
	c.cfg = cfg
	c.window = window

	if err := c.createBG(); err != nil {
		c.log.Error("c.createBG", err)
		return nil
	}

	c.units = append(c.units, runner)

	return &c
}

func (c *Canvas) createBG() error {
	bgImg, err := img.Img("bigbig.jpg", 4096, 4096)
	if err != nil {
		c.log.Error("img.Img", err)
		return nil
	}
	if bgImg == nil {
		c.log.Error("bgImg == nil")
		return nil
	}

	for i := range 64 {
		vector.StrokeLine(
			bgImg,
			0,
			float32(64*i),
			4096,
			float32(64*i),
			2,
			color.RGBA{0xeb, 0xeb, 0xeb, 0xff},
			true,
		)
		vector.StrokeLine(
			bgImg,
			float32(64*i),
			0,
			float32(64*i),
			4096,
			2,
			color.RGBA{0xeb, 0xeb, 0xeb, 0xff},
			true,
		)
	}

	c.bgImage = bgImg
	return nil
}
