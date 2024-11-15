package geom

import (
	"errors"
	"math"
)

type Direction byte

const (
	DirNone Direction = iota
	DirUp
	DirUpRight
	DirRight
	DirDownRight
	DirDown
	DirDownLeft
	DirLeft
	DirUpLeft
)

type Point struct {
	X, Y float64
}

func Pt(x, y float64) Point {
	return Point{x, y}
}

func (p Point) Distance(to Point) (float64, float64) {
	dx := to.X - p.X
	dy := to.Y - p.Y
	return dx, dy
}

func (p Point) Length(to Point) float64 {
	dx := p.X - to.X
	dy := p.Y - to.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (p Point) To(to Point) Direction {
	switch x, y := to.X-p.X, to.Y-p.Y; {
	case x == 0 && y > 0:
		return DirUp
	case x == 0 && y < 0:
		return DirDown
	case x < 0 && y == 0:
		return DirLeft
	case x > 0 && y == 0:
		return DirRight
	case x < 0 && y < 0 && x/y == 1:
		return DirDownLeft
	case x < 0 && y > 0 && x/y == -1:
		return DirUpLeft
	case x > 0 && y < 0 && x/y == -1:
		return DirDownRight
	case x > 0 && y > 0 && x/y == 1:
		return DirUpRight
	default:
		return DirNone
	}
}

// Add returns the vector p+q.
func (p Point) Add(q Point) Point {
	return Point{p.X + q.X, p.Y + q.Y}
}

// Sub returns the vector p-q.
func (p Point) Sub(q Point) Point {
	return Point{p.X - q.X, p.Y - q.Y}
}

// Mul returns the vector p*k.
func (p Point) Mul(k float64) Point {
	return Point{p.X * k, p.Y * k}
}

// Div returns the vector p/k.
func (p Point) Div(k float64) Point {
	return Point{p.X / k, p.Y / k}
}

// In reports whether p is in r.
func (p Point) In(r Rectangle) bool {
	return r.Min.X <= p.X && p.X < r.Max.X &&
		r.Min.Y <= p.Y && p.Y < r.Max.Y
}

// GetNeighbor возвращает соседний Point в заданном направлении
func (p Point) GetNeighbor(dir Direction) (Point, error) {
	switch dir {
	case DirUp:
		return Point{X: p.X, Y: p.Y + 1}, nil
	case DirUpRight:
		return Point{X: p.X + 1, Y: p.Y + 1}, nil
	case DirRight:
		return Point{X: p.X + 1, Y: p.Y}, nil
	case DirDownRight:
		return Point{X: p.X + 1, Y: p.Y - 1}, nil
	case DirDown:
		return Point{X: p.X, Y: p.Y - 1}, nil
	case DirDownLeft:
		return Point{X: p.X - 1, Y: p.Y - 1}, nil
	case DirLeft:
		return Point{X: p.X - 1, Y: p.Y}, nil
	case DirUpLeft:
		return Point{X: p.X - 1, Y: p.Y + 1}, nil
	default:
		return p, errors.New("invalid direction") // если направление некорректно, возвращаем исходную точку
	}
}
