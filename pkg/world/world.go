package world

import (
	"image"
	"image/color"
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

type TileType uint8

const (
	TileGrass TileType = iota
	TileRoad
	TileDirt
	TileSwamp
	TileWater
)

const tileTypeBandSize = 51

func (t TileType) String() string {
	switch t {
	case TileGrass:
		return "grass"
	case TileRoad:
		return "road"
	case TileDirt:
		return "dirt"
	case TileSwamp:
		return "swamp"
	case TileWater:
		return "water"
	default:
		return "unknown"
	}
}

func (t TileType) SpeedMultiplier() float64 {
	switch t {
	case TileRoad:
		return 1.35
	case TileDirt:
		return 0.9
	case TileSwamp:
		return 0.55
	case TileWater:
		return 0.35
	case TileGrass:
		fallthrough
	default:
		return 1
	}
}

func (t TileType) MovementCost() float64 {
	multiplier := t.SpeedMultiplier()
	if multiplier <= 0 {
		return math.Inf(1)
	}

	return 1 / multiplier
}

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

func (w World) InBounds(x, y int) bool {
	return x >= 0 && x < w.columns && y >= 0 && y < w.rows
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

func (w World) TileType(x, y int) TileType {
	if !w.InBounds(x, y) {
		return TileGrass
	}
	if isRoadTile(x, y) {
		return TileRoad
	}

	biome := blendedNoise(x, y)
	switch {
	case biome < 36:
		return TileWater
	case biome < 72:
		return TileSwamp
	case biome > 222:
		return TileDirt
	default:
		return TileGrass
	}
}

func (w World) TileColor(x, y int) color.NRGBA {
	tileType := w.TileType(x, y)
	variant := int(tileHash(x, y) % 23)

	switch tileType {
	case TileRoad:
		return color.NRGBA{R: uint8(116 + variant), G: uint8(106 + variant/2), B: uint8(82 + variant/3), A: 255}
	case TileDirt:
		return color.NRGBA{R: uint8(131 + variant), G: uint8(101 + variant/2), B: uint8(69 + variant/3), A: 255}
	case TileSwamp:
		return color.NRGBA{R: uint8(72 + variant/3), G: uint8(100 + variant/2), B: uint8(68 + variant/4), A: 255}
	case TileWater:
		return color.NRGBA{R: uint8(46 + variant/3), G: uint8(94 + variant/2), B: uint8(142 + variant), A: 255}
	case TileGrass:
		fallthrough
	default:
		return color.NRGBA{R: uint8(78 + variant/4), G: uint8(134 + variant), B: uint8(86 + variant/3), A: 255}
	}
}

func (w World) TileTint(x, y int) color.NRGBA {
	switch w.TileType(x, y) {
	case TileRoad:
		return color.NRGBA{R: 236, G: 222, B: 196, A: 255}
	case TileDirt:
		return color.NRGBA{R: 228, G: 205, B: 178, A: 255}
	case TileSwamp:
		return color.NRGBA{R: 196, G: 218, B: 182, A: 255}
	case TileWater:
		return color.NRGBA{R: 180, G: 210, B: 245, A: 255}
	case TileGrass:
		fallthrough
	default:
		return color.NRGBA{R: 214, G: 232, B: 204, A: 255}
	}
}

func (w World) TileIndex(x, y int) int {
	tileType := w.TileType(x, y)
	base := int(tileType) * tileTypeBandSize
	variant := int(tileHash(x/2, y/2) % tileTypeBandSize)
	return (base + variant) % 256
}

func blendedNoise(x, y int) uint32 {
	coarse := tileHash(x/11, y/11)
	medium := tileHash(x/27, y/27)
	fine := tileHash(x/61, y/61)
	return ((coarse & 0xff) + 2*(medium&0xff) + (fine & 0xff)) / 4
}

func isRoadTile(x, y int) bool {
	return isVerticalRoad(x) || isHorizontalRoad(y)
}

func isVerticalRoad(x int) bool {
	center := verticalRoadCenter(x)
	return absInt(x-center) <= 1
}

func verticalRoadCenter(x int) int {
	const spacing = 72
	band := floorDiv(x, spacing)
	return band*spacing + spacing/2
}

func isHorizontalRoad(y int) bool {
	center := horizontalRoadCenter(y)
	return absInt(y-center) <= 1
}

func horizontalRoadCenter(y int) int {
	const spacing = 96
	band := floorDiv(y, spacing)
	return band*spacing + spacing/2
}

func tileHash(x, y int) uint32 {
	value := uint32(x)*1664525 + uint32(y)*1013904223 + 0x9e3779b9
	value ^= value >> 16
	value *= 0x85ebca6b
	value ^= value >> 13
	return value
}

func floorDiv(value, divisor int) int {
	if divisor <= 0 {
		return 0
	}
	if value >= 0 {
		return value / divisor
	}
	return -((-value + divisor - 1) / divisor)
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
