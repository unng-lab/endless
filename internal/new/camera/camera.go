package camera

import "math"

// Point represents a 2D point in world or screen space.
type Point struct {
	X float64
	Y float64
}

// Rect defines an axis-aligned rectangle in world space.
type Rect struct {
	Min Point
	Max Point
}

// Width returns the rectangle width.
func (r Rect) Width() float64 {
	return r.Max.X - r.Min.X
}

// Height returns the rectangle height.
func (r Rect) Height() float64 {
	return r.Max.Y - r.Min.Y
}

// Camera implements a simple 2D camera with zoom and panning capabilities.
type Camera struct {
	position Point
	scale    float64
	minScale float64
	maxScale float64
}

// Config defines camera construction parameters.
type Config struct {
	// Initial position of the camera in world space.
	Position Point
	// Initial zoom level. If zero, defaults to 1.
	Scale float64
	// Minimal zoom level. If zero, defaults to 0.25.
	MinScale float64
	// Maximal zoom level. If zero, defaults to 4.
	MaxScale float64
}

// New creates a new Camera instance.
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
		scale:    clamp(scale, minScale, maxScale),
		minScale: minScale,
		maxScale: maxScale,
	}
}

// Position returns current world position (top-left corner).
func (c *Camera) Position() Point {
	return c.position
}

// Scale returns current zoom level.
func (c *Camera) Scale() float64 {
	return c.scale
}

// Move shifts the camera by the provided delta in world coordinates.
func (c *Camera) Move(dx, dy float64) {
	c.position.X += dx
	c.position.Y += dy
}

// SetPosition moves the camera to the provided world coordinate.
func (c *Camera) SetPosition(p Point) {
	c.position = p
}

// Zoom changes the current zoom level. The cursor parameter is expected
// to be provided in screen coordinates so that the world position under the
// cursor stays fixed after the zoom.
func (c *Camera) Zoom(delta float64, cursor Point) bool {
	if delta == 0 {
		return false
	}

	newScale := clamp(c.scale*(1+delta), c.minScale, c.maxScale)
	if almostEqual(newScale, c.scale) {
		return false
	}

	worldBefore := c.ScreenToWorld(cursor)

	c.scale = newScale
	c.position.X = worldBefore.X - cursor.X/c.scale
	c.position.Y = worldBefore.Y - cursor.Y/c.scale

	return true
}

// ScreenToWorld converts screen coordinates into world coordinates based on
// current camera state.
func (c *Camera) ScreenToWorld(screen Point) Point {
	return Point{
		X: c.position.X + screen.X/c.scale,
		Y: c.position.Y + screen.Y/c.scale,
	}
}

// ViewRect returns the currently visible world rectangle for the provided
// screen dimensions.
func (c *Camera) ViewRect(screenWidth, screenHeight float64) Rect {
	return Rect{
		Min: c.position,
		Max: Point{
			X: c.position.X + screenWidth/c.scale,
			Y: c.position.Y + screenHeight/c.scale,
		},
	}
}

func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func almostEqual(a, b float64) bool {
	const eps = 1e-9
	return math.Abs(a-b) < eps
}
