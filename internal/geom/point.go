package geom

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
