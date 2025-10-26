// navmesh/navmesh_test.go
package navmesh

import (
	"testing"

	"github.com/unng-lab/madfarmer/internal/chunk"
	"github.com/unng-lab/madfarmer/internal/geom"
)

func TestRectsTouch(t *testing.T) {
	// Случай 1: прямоугольники касаются по горизонтали
	rect1 := &RectPoly{Min: geom.Vec2{0, 0}, Max: geom.Vec2{5, 5}}
	rect2 := &RectPoly{Min: geom.Vec2{6, 0}, Max: geom.Vec2{10, 5}}
	if !RectsTouch(rect1, rect2) {
		t.Error("RectsTouch failed: horizontal touching rectangles should be neighbors")
	}

	// Случай 2: прямоугольники касаются по вертикали
	rect3 := &RectPoly{Min: geom.Vec2{0, 0}, Max: geom.Vec2{5, 5}}
	rect4 := &RectPoly{Min: geom.Vec2{0, 6}, Max: geom.Vec2{5, 10}}
	if !RectsTouch(rect3, rect4) {
		t.Error("RectsTouch failed: vertical touching rectangles should be neighbors")
	}

	// Случай 3: прямоугольники не касаются
	rect5 := &RectPoly{Min: geom.Vec2{0, 0}, Max: geom.Vec2{5, 5}}
	rect6 := &RectPoly{Min: geom.Vec2{7, 7}, Max: geom.Vec2{10, 10}}
	if RectsTouch(rect5, rect6) {
		t.Error("RectsTouch failed: non-touching rectangles should not be neighbors")
	}

	// Случай 4: прямоугольники перекрываются
	rect7 := &RectPoly{Min: geom.Vec2{0, 0}, Max: geom.Vec2{5, 5}}
	rect8 := &RectPoly{Min: geom.Vec2{3, 3}, Max: geom.Vec2{8, 8}}
	if !RectsTouch(rect7, rect8) {
		t.Error("RectsTouch failed: overlapping rectangles should be neighbors")
	}
}

func TestIntervalsOverlap(t *testing.T) {
	// Случай 1: интервалы перекрываются
	if !IntervalsOverlap(0, 5, 3, 8) {
		t.Error("IntervalsOverlap failed: [0,5] and [3,8] should overlap")
	}

	// Случай 2: интервалы касаются
	if !IntervalsOverlap(0, 5, 5, 10) {
		t.Error("IntervalsOverlap failed: [0,5] and [5,10] should be considered overlapping")
	}

	// Случай 3: интервалы не перекрываются
	if IntervalsOverlap(0, 5, 6, 10) {
		t.Error("IntervalsOverlap failed: [0,5] and [6,10] should not overlap")
	}
}

func TestBuildNavMeshFromGrid(t *testing.T) {
	// Создаем простую сетку 4x4 без препятствий
	g := chunk.NewGrid(4, 4)
	id := geom.ChunkID{0, 0}

	mesh := BuildNavMeshFromGrid(g, id)
	if len(mesh.Polys) != 1 {
		t.Errorf("BuildNavMeshFromGrid failed: expected 1 polygon, got %d", len(mesh.Polys))
	}

	poly := mesh.Polys[0]
	if poly.Min.X != 0 || poly.Min.Y != 0 || poly.Max.X != 3 || poly.Max.Y != 3 {
		t.Errorf("BuildNavMeshFromGrid failed: unexpected polygon bounds: %v-%v", poly.Min, poly.Max)
	}

	// Создаем сетку с препятствием в центре
	g = chunk.NewGrid(4, 4)
	g.SetBlocked(1, 1, true)
	g.SetBlocked(1, 2, true)
	g.SetBlocked(2, 1, true)
	g.SetBlocked(2, 2, true)

	mesh = BuildNavMeshFromGrid(g, id)
	if len(mesh.Polys) != 4 {
		t.Errorf("BuildNavMeshFromGrid failed: expected 4 polygons, got %d", len(mesh.Polys))
	}
}
