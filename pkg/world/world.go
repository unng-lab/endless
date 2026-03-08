package world

import (
	"image"
	"image/color"
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

type Config struct {
	Columns  int
	Rows     int
	TileSize float64
}

type World struct {
	columns  int
	rows     int
	tileSize float64
}

func New(cfg Config) World {
	return World{
		columns:  cfg.Columns,
		rows:     cfg.Rows,
		tileSize: cfg.TileSize,
	}
}

func (w World) Columns() int {
	return w.columns
}

func (w World) Rows() int {
	return w.rows
}

func (w World) TileSize() float64 {
	return w.tileSize
}

func (w World) Width() float64 {
	return float64(w.columns) * w.tileSize
}

func (w World) Height() float64 {
	return float64(w.rows) * w.tileSize
}

func (w World) VisibleRange(view geom.Rect) image.Rectangle {
	minX := geom.ClampInt(int(math.Floor(view.Min.X/w.tileSize)), 0, w.columns)
	minY := geom.ClampInt(int(math.Floor(view.Min.Y/w.tileSize)), 0, w.rows)
	maxX := geom.ClampInt(int(math.Ceil(view.Max.X/w.tileSize))+1, 0, w.columns)
	maxY := geom.ClampInt(int(math.Ceil(view.Max.Y/w.tileSize))+1, 0, w.rows)

	return image.Rect(minX, minY, maxX, maxY)
}

func (w World) ClampCamera(pos geom.Point, scale float64, screenWidth, screenHeight int) geom.Point {
	viewWidth := float64(screenWidth) / scale
	viewHeight := float64(screenHeight) / scale
	maxX := math.Max(w.Width()-viewWidth, 0)
	maxY := math.Max(w.Height()-viewHeight, 0)

	pos.X = geom.ClampFloat(pos.X, 0, maxX)
	pos.Y = geom.ClampFloat(pos.Y, 0, maxY)

	return pos
}

func TileGap(tileScreenSize float64) float64 {
	switch {
	case tileScreenSize >= 28:
		return 2
	case tileScreenSize >= 12:
		return 1
	default:
		return 0
	}
}

func TileColor(x, y int) color.NRGBA {
	switch {
	case (x+y)%17 == 0:
		return color.NRGBA{R: 165, G: 121, B: 73, A: 255}
	case (x/8+y/8)%2 == 0:
		return color.NRGBA{R: 69, G: 118, B: 82, A: 255}
	case (x+y)%2 == 0:
		return color.NRGBA{R: 78, G: 138, B: 93, A: 255}
	default:
		return color.NRGBA{R: 95, G: 148, B: 108, A: 255}
	}
}

func TileIndex(x, y int) int {
	blockX := x / 4
	blockY := y / 4

	value := uint32(blockX)*1664525 + uint32(blockY)*1013904223 + 0x9e3779b9
	value ^= value >> 16
	return int(value % 256)
}
