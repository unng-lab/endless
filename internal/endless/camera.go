package endless

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	atomic2 "go.uber.org/atomic"
)

const (
	maxZoom = 200
	minZoom = -100
)

var C = Camera{
	tileSize:    *atomic2.NewFloat64(TileSize),
	scaleFactor: *atomic2.NewFloat64(1),
}

type Camera struct {
	positionX   atomic2.Float64
	positionY   atomic2.Float64
	zoomFactor  atomic2.Float64
	tileSize    atomic2.Float64
	scaleFactor atomic2.Float64
}

func (c *Camera) String() string {
	return fmt.Sprintf(
		"X: %d,Y: %d, S: %d",
		c.positionX.Load(), c.positionY.Load(), c.zoomFactor.Load(),
	)
}

func (c *Camera) worldMatrix() ebiten.GeoM {
	m := ebiten.GeoM{}
	m.Translate(-c.positionX.Load(), -c.positionY.Load())
	// We want to scale and rotate around center of image / screen
	m.Translate(W.ViewPortCenter(false))
	m.Scale(
		math.Pow(1.01, c.zoomFactor.Load()),
		math.Pow(1.01, c.zoomFactor.Load()),
	)
	return m
}

func (c *Camera) Render(world, screen *ebiten.Image) {
	screen.DrawImage(world, &ebiten.DrawImageOptions{
		GeoM: c.worldMatrix(),
	})
}

func (c *Camera) ScreenToWorld(posX, posY int) (float64, float64) {
	inverseMatrix := c.worldMatrix()
	if inverseMatrix.IsInvertible() {
		inverseMatrix.Invert()
		return inverseMatrix.Apply(float64(posX), float64(posY))
	} else {
		// When scaling it can happened that matrix is not invertable
		return math.NaN(), math.NaN()
	}
}

func (c *Camera) Reset(w, h int) {
	c.positionX.Store(0)
	c.positionY.Store(0)
	c.zoomFactor.Store(0)
	c.tileSize.Store(TileSize * c.scale())
	c.scaleFactor.Store(c.scale())
}

func (c *Camera) Up() {
	c.positionY.Add(-50)
}

func (c *Camera) Down() {
	c.positionY.Add(50)
}

func (c *Camera) Left() {
	c.positionX.Add(-50)
}

func (c *Camera) Right() {
	c.positionX.Add(50)
}

func (c *Camera) ZoomUp() {
	if c.zoomFactor.Load() < maxZoom {
		c.zoomFactor.Add(10)
		c.tileSize.Store(TileSize * c.scale())
		c.scaleFactor.Store(c.scale())
	}
}

func (c *Camera) ZoomDown() {
	if c.zoomFactor.Load() > minZoom {
		c.zoomFactor.Add(-10)
		c.tileSize.Store(TileSize * c.scale())
		c.scaleFactor.Store(c.scale())
	}
}

func (c *Camera) scale() float64 {
	return math.Pow(1.01, c.zoomFactor.Load())
}

func (c *Camera) GetTileSize() float64 {
	return c.tileSize.Load()
}

func (c *Camera) GetZoomFactor() float64 {
	return c.zoomFactor.Load()
}

func (c *Camera) GetPositionX() float64 {
	return c.positionX.Load()
}

func (c *Camera) GetPositionY() float64 {
	return c.positionY.Load()
}

func (c *Camera) GetScaleFactor() float64 {
	return c.scaleFactor.Load()
}
