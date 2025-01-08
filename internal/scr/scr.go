package scr

import (
	"fmt"
	"image"
	"log/slog"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/unng-lab/madfarmer/assets/img"
	"github.com/unng-lab/madfarmer/internal/ebitenfx"
	"github.com/unng-lab/madfarmer/internal/units/runner"
	"github.com/unng-lab/madfarmer/internal/window"
)

var _ ebitenfx.Screen = (*Canvas)(nil)

const (
	mapBlockSize  = 2048
	mapBlockCount = 5
)

type Config interface {
}
type Canvas struct {
	log    *slog.Logger
	cfg    Config
	window *window.Default
	camera Camera
	//TMP
	terrain *ebiten.Image
	bg      *ebiten.Image
	counter int64
	units   []Unit

	mapTiles map[string]*ebiten.Image
}

type Unit interface {
	Position() image.Point
	Draw(
		bg *ebiten.Image,
		counter int64,
		div image.Point,
		min image.Point,
		max image.Point,
	)
}

func (c *Canvas) Draw(screen *ebiten.Image) {
	w, h := mapBlockSize, mapBlockSize
	dX := int(c.camera.position[0]) / w
	dY := int(c.camera.position[1]) / h
	for j := range mapBlockCount {
		for i := range mapBlockCount {
			key := fmt.Sprintf("%d,%d",
				int(math.Abs(float64((i+dX)%mapBlockCount))),
				int(math.Abs(float64((j+dY)%mapBlockCount))))
			if tile, ok := c.mapTiles[key]; !ok {
				c.log.Error("tile not found", i, j)
				return
			} else {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(w*i), float64(h*j))
				c.bg.DrawImage(tile, op)
			}
		}
	}
	//if c.camera.zoomFactor >= 0 {
	//	for i := range 64 {
	//		vector.StrokeLine(
	//			c.bg,
	//			0,
	//			float32(64*i),
	//			mapBlockSize,
	//			float32(64*i),
	//			2,
	//			color.RGBA{0xeb, 0xeb, 0xeb, 0xff},
	//			true,
	//		)
	//		vector.StrokeLine(
	//			c.bg,
	//			float32(64*i),
	//			0,
	//			float32(64*i),
	//			mapBlockSize,
	//			2,
	//			color.RGBA{0xeb, 0xeb, 0xeb, 0xff},
	//			true,
	//		)
	//	}
	//}

	for _, unit := range c.units {
		unit.Draw(
			c.bg,
			c.counter,
			image.Pt(w, h),
			image.Pt((dX-mapBlockCount/2)*mapBlockSize, (dY-mapBlockCount/2)*mapBlockSize),
			image.Pt((dX+mapBlockCount/2+1)*mapBlockSize, (dY+mapBlockCount/2+1)*mapBlockSize),
		)
	}

	options := &ebiten.DrawImageOptions{
		GeoM: c.camera.worldMatrix(),
	}

	//options.GeoM.Translate(float64(-mapBlockSize),
	//	float64(-mapBlockSize))
	c.log.Info("geoM", dX, dY)
	screen.DrawImage(c.bg, options)

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

	c.units = append(c.units, runner)
	c.mapTiles = make(map[string]*ebiten.Image)
	c.bg = ebiten.NewImage(mapBlockCount*mapBlockSize, mapBlockCount*mapBlockSize)
	c.camera.window = window
	if err := c.createBG(); err != nil {
		c.log.Error("c.createBG", err)
		return nil
	}

	return &c
}

func (c *Canvas) createBG() error {
	for j := range mapBlockCount {
		for i := range mapBlockCount {
			terrain, err := img.Img(getNameBg(), mapBlockSize, mapBlockSize)
			if err != nil {
				c.log.Error("img.Img", err)
				return nil
			}
			if terrain == nil {
				c.log.Error("terrain == nil")
				return nil
			}
			c.mapTiles[fmt.Sprintf("%d,%d", i, j)] = terrain
		}
	}

	return nil
}

var nArr = []string{
	"bigbig.jpg",
	"bigterrain.jpg",
	"terrain.jpg",
}

func getNameBg() string {
	return nArr[rand.Intn(len(nArr))]
}
