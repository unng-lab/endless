package camera

import (
	"math"
	"testing"
)

func TestZoomKeepsCursorWorldPosition(t *testing.T) {
	cam := New(Config{})
	cam.SetPosition(Point{X: 100, Y: 50})

	cursor := Point{X: 320, Y: 240}
	worldBefore := cam.ScreenToWorld(cursor)

	changed := cam.Zoom(0.5, cursor)
	if !changed {
		t.Fatalf("expected zoom to change scale")
	}

	worldAfter := cam.ScreenToWorld(cursor)

	if diff := distance(worldBefore, worldAfter); diff > 1e-6 {
		t.Fatalf("cursor world position drifted after zoom: diff=%f", diff)
	}
}

func TestZoomClampsToBounds(t *testing.T) {
	cam := New(Config{Scale: 1, MinScale: 0.5, MaxScale: 2})

	cam.Zoom(-0.9, Point{})
	if got := cam.Scale(); !almostEqual(got, 0.5) {
		t.Fatalf("expected min scale 0.5, got %f", got)
	}

	cam.Zoom(10, Point{})
	if got := cam.Scale(); !almostEqual(got, 2) {
		t.Fatalf("expected max scale 2, got %f", got)
	}
}

func TestViewRect(t *testing.T) {
	cam := New(Config{Position: Point{X: 10, Y: 20}, Scale: 2})
	rect := cam.ViewRect(200, 100)

	expected := Rect{
		Min: Point{X: 10, Y: 20},
		Max: Point{X: 10 + 200/2, Y: 20 + 100/2},
	}

	if !rectEqual(rect, expected) {
		t.Fatalf("unexpected view rect: %#v", rect)
	}
}

func distance(a, b Point) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Hypot(dx, dy)
}

func rectEqual(a, b Rect) bool {
	return almostEqual(a.Min.X, b.Min.X) &&
		almostEqual(a.Min.Y, b.Min.Y) &&
		almostEqual(a.Max.X, b.Max.X) &&
		almostEqual(a.Max.Y, b.Max.Y)
}
