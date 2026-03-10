package unit

import (
	"testing"

	"github.com/unng-lab/endless/pkg/geom"
)

// TestOrderedUnitMapPreservesInsertionOrderOnReplacement verifies that replacing the runtime
// object for an already known UnitID updates direct lookups in place and keeps the established
// iteration order stable for worker traversal and lifecycle passes.
func TestOrderedUnitMapPreservesInsertionOrderOnReplacement(t *testing.T) {
	units := newOrderedUnitMap(2)
	first := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	first.SetUnitID(10)
	second := NewRunner(geom.Point{X: 24, Y: 24}, false, 0)
	second.SetUnitID(20)
	replacement := NewRunner(geom.Point{X: 40, Y: 40}, true, 0)
	replacement.SetUnitID(10)

	units.Set(first)
	units.Set(second)
	units.Set(replacement)

	if units.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", units.Len())
	}

	gotByID, ok := units.Get(10)
	if !ok {
		t.Fatal("Get(10) = false, want replacement unit")
	}
	if gotByID != replacement {
		t.Fatalf("Get(10) returned %p, want replacement %p", gotByID, replacement)
	}

	gotFirst, ok := units.At(0)
	if !ok {
		t.Fatal("At(0) = false, want replacement unit")
	}
	if gotFirst != replacement {
		t.Fatalf("At(0) returned %p, want replacement %p", gotFirst, replacement)
	}

	gotSecond, ok := units.At(1)
	if !ok {
		t.Fatal("At(1) = false, want second unit")
	}
	if gotSecond != second {
		t.Fatalf("At(1) returned %p, want second %p", gotSecond, second)
	}
}

// TestOrderedUnitMapRangeStopsAtVisitorRequest verifies that Range still walks units in
// insertion order and honors an early-stop request from the visitor without traversing the
// remaining dense storage.
func TestOrderedUnitMapRangeStopsAtVisitorRequest(t *testing.T) {
	units := newOrderedUnitMap(3)
	first := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	first.SetUnitID(1)
	second := NewRunner(geom.Point{X: 24, Y: 24}, false, 0)
	second.SetUnitID(2)
	third := NewRunner(geom.Point{X: 40, Y: 40}, false, 0)
	third.SetUnitID(3)

	units.Set(first)
	units.Set(second)
	units.Set(third)

	visited := make([]int64, 0, 2)
	units.Range(func(unit Unit) bool {
		visited = append(visited, unit.UnitID())
		return unit.UnitID() != 2
	})

	if len(visited) != 2 {
		t.Fatalf("visited %d units, want 2", len(visited))
	}
	if visited[0] != 1 || visited[1] != 2 {
		t.Fatalf("visited order = %v, want [1 2]", visited)
	}
}
