package geom

import "math"

type Point struct {
	X float64
	Y float64
}

type Rect struct {
	Min Point
	Max Point
}

func ClampFloat(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func ClampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func AlmostEqual(a, b float64) bool {
	const eps = 1e-9
	return math.Abs(a-b) < eps
}
