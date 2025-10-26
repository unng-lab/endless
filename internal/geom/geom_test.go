// geom/geom_test.go
package geom

import (
	"testing"
)

func TestVec2(t *testing.T) {
	v1 := Vec2{3, 4}
	v2 := Vec2{1, 2}

	// Test Add
	result := v1.Add(v2)
	if result.X != 4 || result.Y != 6 {
		t.Errorf("Add failed: expected (4,6), got (%d,%d)", result.X, result.Y)
	}

	// Test Eq
	if v1.Eq(v2) {
		t.Error("Eq failed: v1 should not equal v2")
	}

	v3 := Vec2{3, 4}
	if !v1.Eq(v3) {
		t.Error("Eq failed: v1 should equal v3")
	}
}

func TestWorldToChunk(t *testing.T) {
	const ChunkSize = 32

	// Test positive coordinates
	pos := Vec2{40, 50}
	chunkID, local := WorldToChunk(pos, ChunkSize)
	if chunkID.X != 1 || chunkID.Y != 1 || local.X != 8 || local.Y != 18 {
		t.Errorf("WorldToChunk failed for positive coords: expected chunk (1,1) local (8,18), got chunk (%d,%d) local (%d,%d)",
			chunkID.X, chunkID.Y, local.X, local.Y)
	}

	// Test negative coordinates
	pos = Vec2{-10, -20}
	chunkID, local = WorldToChunk(pos, ChunkSize)
	if chunkID.X != -1 || chunkID.Y != -1 || local.X != 22 || local.Y != 12 {
		t.Errorf("WorldToChunk failed for negative coords: expected chunk (-1,-1) local (22,12), got chunk (%d,%d) local (%d,%d)",
			chunkID.X, chunkID.Y, local.X, local.Y)
	}
}

func TestFloorDiv(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{10, 3, 3},
		{-10, 3, -4},
		{10, -3, -4},
		{-10, -3, 3},
		{0, 5, 0},
		{5, 5, 1},
		{-5, 5, -1},
	}

	for _, test := range tests {
		result := floorDiv(test.a, test.b)
		if result != test.expected {
			t.Errorf("floorDiv(%d, %d) = %d, expected %d", test.a, test.b, result, test.expected)
		}
	}
}

func TestDist2(t *testing.T) {
	a := Vec2{0, 0}
	b := Vec2{3, 4}

	dist := Dist2(a, b)
	if dist != 25 {
		t.Errorf("Dist2 failed: expected 25, got %d", dist)
	}
}
