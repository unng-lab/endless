package tilemap

import (
	"image"
	"testing"

	"github.com/unng-lab/endless/internal/new/camera"
)

func TestVisibleRangeWithinBounds(t *testing.T) {
	m := New(Config{Columns: 10, Rows: 8, TileSize: 32})
	cam := camera.New(camera.Config{Position: camera.Point{X: 0, Y: 0}, Scale: 1})

	rect := m.VisibleRange(cam, 128, 128)
	expected := image.Rect(0, 0, 4, 4)

	if rect != expected {
		t.Fatalf("expected %v, got %v", expected, rect)
	}
}

func TestVisibleRangeClampedToMap(t *testing.T) {
	m := New(Config{Columns: 5, Rows: 5, TileSize: 32})
	cam := camera.New(camera.Config{Position: camera.Point{X: 32 * 3, Y: 32 * 3}, Scale: 1})

	rect := m.VisibleRange(cam, 200, 200)
	expected := image.Rect(3, 3, 5, 5)

	if rect != expected {
		t.Fatalf("expected %v, got %v", expected, rect)
	}
}
