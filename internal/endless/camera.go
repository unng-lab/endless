package endless

import (
	"math"
)

const (
	maxZoom = 800
	minZoom = -100
)

// Camera TODO add maxX and maxY camera
type Camera struct {
	positionX  float64
	positionY  float64
	zoomFactor float64
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
