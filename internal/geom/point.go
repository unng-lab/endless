package geom

import (
	"math"
)

const (
	DirUp byte = iota
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

func (p Point) To(to Point) byte {
	switch x, y := to.X-p.X, to.Y-p.Y; {
	case x == 0 && y < 0:
		return DirUp
	case x == 0 && y > 0:
		return DirDown
	case x < 0 && y == 0:
		return DirLeft
	case x > 0 && y == 0:
		return DirRight
	case x < 0 && y < 0:
		return DirUpLeft
	case x < 0 && y > 0:
		return DirDownLeft
	case x > 0 && y < 0:
		return DirUpRight
	case x > 0 && y > 0:
		return DirDownRight
	default:
		panic("unreachable")
	}
}
