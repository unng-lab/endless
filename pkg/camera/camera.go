package camera

import "github.com/unng-lab/endless/pkg/geom"

type Config struct {
	Position geom.Point
	Scale    float64
	MinScale float64
	MaxScale float64
}

type Camera struct {
	position geom.Point
	scale    float64
	minScale float64
	maxScale float64
}

func New(cfg Config) *Camera {
	scale := cfg.Scale
	if scale == 0 {
		scale = 1
	}
	minScale := cfg.MinScale
	if minScale == 0 {
		minScale = 0.25
	}
	maxScale := cfg.MaxScale
	if maxScale == 0 {
		maxScale = 4
	}

	return &Camera{
		position: cfg.Position,
		scale:    geom.ClampFloat(scale, minScale, maxScale),
		minScale: minScale,
		maxScale: maxScale,
	}
}

func (c *Camera) Position() geom.Point {
	return c.position
}

func (c *Camera) SetPosition(pos geom.Point) {
	c.position = pos
}

func (c *Camera) Move(dx, dy float64) {
	c.position.X += dx
	c.position.Y += dy
}

func (c *Camera) Scale() float64 {
	return c.scale
}

func (c *Camera) Zoom(delta float64, cursor geom.Point) bool {
	if delta == 0 {
		return false
	}

	newScale := geom.ClampFloat(c.scale*(1+delta), c.minScale, c.maxScale)
	if geom.AlmostEqual(newScale, c.scale) {
		return false
	}

	worldBefore := c.ScreenToWorld(cursor)
	c.scale = newScale
	c.position.X = worldBefore.X - cursor.X/c.scale
	c.position.Y = worldBefore.Y - cursor.Y/c.scale

	return true
}

func (c *Camera) ScreenToWorld(screen geom.Point) geom.Point {
	return geom.Point{
		X: c.position.X + screen.X/c.scale,
		Y: c.position.Y + screen.Y/c.scale,
	}
}

func (c *Camera) ViewRect(screenWidth, screenHeight float64) geom.Rect {
	return geom.Rect{
		Min: c.position,
		Max: geom.Point{
			X: c.position.X + screenWidth/c.scale,
			Y: c.position.Y + screenHeight/c.scale,
		},
	}
}
