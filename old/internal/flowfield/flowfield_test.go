// flowfield/flowfield_test.go
package flowfield

import (
	"testing"

	"github.com/unng-lab/endless/internal/chunk"
	"github.com/unng-lab/endless/internal/geom"
)

func manhattan(a, b geom.Vec2) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

func assertDirectionTowardsTarget(t *testing.T, field map[geom.Vec2]geom.Vec2, pos, target, min, max geom.Vec2, blocked map[geom.Vec2]bool) {
	t.Helper()
	dir, ok := field[pos]
	if !ok {
		t.Fatalf("direction for %v not found", pos)
	}
	if dir.X == 0 && dir.Y == 0 {
		t.Fatalf("direction for %v is zero", pos)
	}
	next := pos.Add(dir)
	if next.X < min.X || next.Y < min.Y || next.X > max.X || next.Y > max.Y {
		t.Fatalf("direction from %v leads outside bounds to %v", pos, next)
	}
	if blocked != nil && blocked[next] {
		t.Fatalf("direction from %v leads into blocked cell %v", pos, next)
	}
	if manhattan(next, target) >= manhattan(pos, target) {
		t.Fatalf("direction from %v does not move closer to target: next=%v", pos, next)
	}
}

func TestBuildFlowField(t *testing.T) {
	// Создаем простую область 3x3
	min := geom.Vec2{0, 0}
	max := geom.Vec2{2, 2}

	// Создаем ChunkManager с простым чанком
	cm := chunk.NewChunkManager()

	// Строим поле потока
	field, err := BuildFlowField(cm, min, max)
	if err != nil {
		t.Fatalf("BuildFlowField failed: %v", err)
	}

	// Проверяем размер
	expectedSize := 9                 // 3x3
	if len(field) != expectedSize-1 { // цель не включается
		t.Errorf("BuildFlowField failed: expected %d cells, got %d", expectedSize-1, len(field))
	}

	// Проверяем направления к цели (2,2)
	target := geom.Vec2{2, 2}
	minBounds := geom.Vec2{0, 0}
	maxBounds := geom.Vec2{2, 2}
	assertDirectionTowardsTarget(t, field, geom.Vec2{1, 2}, target, minBounds, maxBounds, nil)
	assertDirectionTowardsTarget(t, field, geom.Vec2{2, 1}, target, minBounds, maxBounds, nil)
	assertDirectionTowardsTarget(t, field, geom.Vec2{1, 1}, target, minBounds, maxBounds, nil)
}

func TestBuildFlowFieldWithObstacle(t *testing.T) {
	// Создаем область 3x3
	min := geom.Vec2{0, 0}
	max := geom.Vec2{2, 2}

	// Создаем ChunkManager с препятствием
	cm := chunk.NewChunkManager()
	chunkID := geom.ChunkID{0, 0}
	c := cm.EnsureLoaded(chunkID)
	c.Grid.SetBlocked(1, 1, true) // препятствие в центре

	// Строим поле потока
	field, err := BuildFlowField(cm, min, max)
	if err != nil {
		t.Fatalf("BuildFlowField failed: %v", err)
	}

	// Проверяем, что центральная ячейка отсутствует
	if _, ok := field[geom.Vec2{1, 1}]; ok {
		t.Error("BuildFlowField failed: obstacle cell should not be in field")
	}

	// Проверяем обход препятствия
	blocked := map[geom.Vec2]bool{{1, 1}: true}
	target := geom.Vec2{2, 2}
	minBounds := geom.Vec2{0, 0}
	maxBounds := geom.Vec2{2, 2}
	assertDirectionTowardsTarget(t, field, geom.Vec2{1, 0}, target, minBounds, maxBounds, blocked)
	assertDirectionTowardsTarget(t, field, geom.Vec2{0, 1}, target, minBounds, maxBounds, blocked)
}
