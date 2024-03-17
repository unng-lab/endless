package endless

import (
	"math"

	"github/unng-lab/madfarmer/internal/geom"
)

const (
	maxZoom = 200
	minZoom = -100
)

// Camera TODO add maxX and maxY camera
type Camera struct {
	positionX   float64
	positionY   float64
	zoomFactor  float64
	TileSize    float64
	Coordinates geom.Rectangle
	Pixels      geom.Rectangle
}

func (c *Camera) Reset(w, h int) {
	c.positionX = 0
	c.positionY = 0
	c.zoomFactor = 0
}

func (c *Camera) Up() {
	c.positionY += -50
}

func (c *Camera) Down() {
	c.positionY += 50
}

func (c *Camera) Left() {
	c.positionX += -50
}

func (c *Camera) Right() {
	c.positionX += 50
}

func (c *Camera) ZoomUp() {
	if c.zoomFactor < maxZoom {
		c.zoomFactor += 10
	}
}

func (c *Camera) ZoomDown() {
	if c.zoomFactor > minZoom {
		c.zoomFactor += -10
	}
}

func (c *Camera) scale() float64 {
	return math.Pow(1.01, c.zoomFactor)
}

func (c *Camera) GetTileSize() float64 {
	return TileSize * c.scale()
}

func (c *Camera) GetZoomFactor() float64 {
	return c.zoomFactor
}

func (c *Camera) GetPositionX() float64 {
	return c.positionX
}

func (c *Camera) GetPositionY() float64 {
	return c.positionY
}

func (c *Camera) GetScaleFactor() float64 {
	return c.scale()
}

func (c *Camera) GetCurrentCoordinates() geom.Rectangle {
	return c.Coordinates
}

func (c *Camera) Prepare() {
	c.TileSize = c.GetTileSize()
	maxX, maxY := (W.GetWidth())/c.TileSize+1, (W.GetHeight())/c.TileSize+1

	var (
		x, y         float64
		cellX, cellY float64 = CountTile / 2, CountTile / 2
	)

	shiftX, shiftY := math.Mod(c.positionX, c.TileSize), math.Mod(c.positionY, c.TileSize)
	if shiftX < 0 {
		x = -c.TileSize - shiftX
		cellX += -1
	} else if shiftX > 0 {
		x = -shiftX
	}
	cellX += math.Trunc(c.positionX / c.TileSize)

	if shiftY < 0 {
		y = -c.TileSize - shiftY
		cellY += -1
	} else if shiftY > 0 {
		y = -shiftY
	}
	cellY += math.Trunc(c.positionY / c.TileSize)

	c.Coordinates = geom.Rectangle{
		Min: geom.Point{
			X: cellX,
			Y: cellY,
		},
		Max: geom.Point{
			X: cellX + maxX,
			Y: cellY + maxY,
		},
	}

	c.Pixels = geom.Rectangle{
		Min: geom.Point{
			X: math.Round(x*100) / 100,
			Y: math.Round(y*100) / 100,
		},
		Max: geom.Point{
			X: math.Round(x*100)/100 + c.TileSize*(cellX+maxX),
			Y: math.Round(y*100)/100 + c.TileSize*(cellY+maxY),
		},
	}
}

func (c *Camera) GetCurrentPixels() geom.Rectangle {
	return c.Pixels
}

func (c *Camera) GetMiddleInPixels(point geom.Point) geom.Point {
	distX, distY := c.Coordinates.Min.Distance(point)
	return geom.Point{
		X: c.Pixels.Min.X + distX*c.TileSize,
		Y: c.Pixels.Min.Y + distY*c.TileSize,
	}
}
