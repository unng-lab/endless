package tilemap

import (
	"image"
	"math"

	"github.com/unng-lab/endless/internal/new/camera"
)

// TileMap represents a simple grid of boolean tiles.
type TileMap struct {
	columns  int
	rows     int
	tileSize float64
	tiles    []bool
}

// Config contains map creation parameters.
type Config struct {
	Columns  int
	Rows     int
	TileSize float64
}

// New creates a TileMap with a checkerboard pattern.
func New(cfg Config) *TileMap {
	columns := cfg.Columns
	if columns <= 0 {
		columns = 128
	}
	rows := cfg.Rows
	if rows <= 0 {
		rows = 128
	}
	tileSize := cfg.TileSize
	if tileSize <= 0 {
		tileSize = 32
	}

	tiles := make([]bool, columns*rows)
	for y := 0; y < rows; y++ {
		for x := 0; x < columns; x++ {
			if (x+y)%2 == 0 {
				tiles[y*columns+x] = true
			}
		}
	}

	return &TileMap{
		columns:  columns,
		rows:     rows,
		tileSize: tileSize,
		tiles:    tiles,
	}
}

// Columns returns the number of columns in the map.
func (m *TileMap) Columns() int {
	return m.columns
}

// Rows returns the number of rows in the map.
func (m *TileMap) Rows() int {
	return m.rows
}

// TileSize returns the tile size in world units.
func (m *TileMap) TileSize() float64 {
	return m.tileSize
}

// TileAt returns the boolean value of a tile.
func (m *TileMap) TileAt(x, y int) bool {
	if x < 0 || x >= m.columns || y < 0 || y >= m.rows {
		return false
	}
	return m.tiles[y*m.columns+x]
}

// VisibleRange calculates tile indices visible for the current camera.
func (m *TileMap) VisibleRange(cam *camera.Camera, screenWidth, screenHeight int) image.Rectangle {
	rect := cam.ViewRect(float64(screenWidth), float64(screenHeight))
	minX := clampInt(int(math.Floor(rect.Min.X/m.tileSize)), 0, m.columns)
	minY := clampInt(int(math.Floor(rect.Min.Y/m.tileSize)), 0, m.rows)
	maxX := clampInt(int(math.Ceil(rect.Max.X/m.tileSize)), 0, m.columns)
	maxY := clampInt(int(math.Ceil(rect.Max.Y/m.tileSize)), 0, m.rows)

	return image.Rect(minX, minY, maxX, maxY)
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
