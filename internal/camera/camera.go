package camera

import (
	"math"

	"github/unng-lab/madfarmer/internal/geom"
	"github/unng-lab/madfarmer/internal/window"
)

const (
	MaxZoom = 200
	MinZoom = -100
)

var DefaultTileSize float64
var CountTile float64

// Camera TODO add maxX and maxY camera
type Camera struct {
	positionX   float64
	positionY   float64
	zoomFactor  float64
	TileSize    float64
	Coordinates geom.Rectangle
	Pixels      geom.Rectangle
	DrawArea    geom.Rectangle
	ScaleFactor float64
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
	if c.zoomFactor < MaxZoom {
		c.zoomFactor += 10
	}
}

func (c *Camera) ZoomDown() {
	if c.zoomFactor > MinZoom {
		c.zoomFactor += -10
	}
}

func (c *Camera) scale() float64 {
	return math.Pow(1.01, c.zoomFactor)
}

func (c *Camera) GetTileSize() float64 {
	return DefaultTileSize * c.ScaleFactor
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
	return c.ScaleFactor
}

func (c *Camera) GetCurrentCoordinates() geom.Rectangle {
	return c.Coordinates
}

func (c *Camera) Prepare() {
	c.ScaleFactor = c.scale()
	c.TileSize = c.GetTileSize()
	maxX, maxY := (window.W.GetWidth())/c.TileSize+1, (window.W.GetHeight())/c.TileSize+1

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
			X: math.Round(c.TileSize*cellX*100)/100 - x - c.positionX,
			Y: math.Round(c.TileSize*cellY*100)/100 - y - c.positionY,
		},
		Max: geom.Point{
			X: math.Round(c.TileSize*(cellX+maxX)*100)/100 - x - c.positionX,
			Y: math.Round(c.TileSize*(cellY+maxY)*100)/100 - y - c.positionY,
		},
	}
	c.DrawArea = geom.Rectangle{
		Min: geom.Point{
			X: x,
			Y: y,
		},
		Max: geom.Point{
			X: x + c.TileSize*maxX,
			Y: y + c.TileSize*maxY,
		},
	}
}

func (c *Camera) GetCurrentPixels() geom.Rectangle {
	return c.Pixels
}

func (c *Camera) GetMiddleInPixels(point geom.Point) geom.Point {
	distX, distY := c.Coordinates.Min.Distance(point)
	return geom.Point{
		X: c.DrawArea.Min.X + distX*c.TileSize + c.TileSize/2,
		Y: c.DrawArea.Min.Y + distY*c.TileSize + c.TileSize/2,
	}
}

//func (c *Camera) DrawPixelMinPoint() (float64, float64) {
//	return c.Pixels.Min.X - c.TileSize*CountTile/2,
//		c.Pixels.Min.Y - c.TileSize*CountTile/2
//}
