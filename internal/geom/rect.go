package geom

type Rectangle struct {
	Min, Max Point
}

func Rect(x, y, w, h float64) Rectangle {
	return Rectangle{Pt(x, y), Pt(x+w, y+h)}
}

func (r Rectangle) Contains(p Point) bool {
	return p.X >= r.Min.X && p.X <= r.Max.X && p.Y >= r.Min.Y && p.Y <= r.Max.Y
}

func (r Rectangle) ContainsOR(p Point) bool {
	return p.X <= r.Min.X || p.X >= r.Max.X || p.Y <= r.Min.Y || p.Y >= r.Max.Y
}
