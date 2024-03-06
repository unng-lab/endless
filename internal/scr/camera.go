package scr

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/math/f64"
)

type Camera struct {
	viewPort   f64.Vec2
	position   f64.Vec2
	zoomFactor int
	rotation   int
	middleX    int
	middleY    int
}

func (c *Camera) String() string {
	return fmt.Sprintf(
		"T: %.1f, R: %d, S: %d",
		c.position, c.rotation, c.zoomFactor,
	)
}

func (c *Camera) viewportCenter() f64.Vec2 {
	return f64.Vec2{
		c.viewPort[0] * 0.5,
		c.viewPort[1] * 0.5,
	}
}

func (c *Camera) worldMatrix() ebiten.GeoM {
	m := ebiten.GeoM{}
	m.Translate(-c.position[0], -c.position[1])
	// We want to scale and rotate around center of image / screen
	m.Translate(-c.viewportCenter()[0], -c.viewportCenter()[1])
	m.Scale(
		math.Pow(1.01, float64(c.zoomFactor)),
		math.Pow(1.01, float64(c.zoomFactor)),
	)
	m.Rotate(float64(c.rotation) * 2 * math.Pi / 360)
	m.Translate(c.viewportCenter()[0], c.viewportCenter()[1])
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
	c.position[0] = float64(c.middleX - w/2)
	c.position[1] = float64(c.middleY - h/2)
	c.rotation = 0
	c.zoomFactor = 0
}

func (c *Camera) Up() {
	c.position[1] -= 50
}

func (c *Camera) Down() {
	c.position[1] += 50
}

func (c *Camera) Left() {
	c.position[0] -= 50
}

func (c *Camera) Right() {
	c.position[0] += 50
}

func (c *Camera) ZoomUp() {
	if c.zoomFactor < 2400 {
		c.zoomFactor += 10
	}
}

func (c *Camera) ZoomDown() {
	if c.zoomFactor > -2400 {
		c.zoomFactor -= 10
	}
}

func (c *Camera) SetMiddle(x, y int) {

}

func (c *Camera) Rotation(v int) {
	c.rotation += v
}
