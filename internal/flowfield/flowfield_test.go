// flowfield/flowfield_test.go
package flowfield

import (
	"testing"

	"github.com/unng-lab/madfarmer/internal/chunk"
	"github.com/unng-lab/madfarmer/internal/geom"
)

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
	if dir, ok := field[geom.Vec2{1, 2}]; !ok || dir.X != 1 || dir.Y != 0 {
		t.Errorf("BuildFlowField failed: wrong direction for (1,2), got %v", dir)
	}
	if dir, ok := field[geom.Vec2{2, 1}]; !ok || dir.X != 0 || dir.Y != 1 {
		t.Errorf("BuildFlowField failed: wrong direction for (2,1), got %v", dir)
	}
	if dir, ok := field[geom.Vec2{1, 1}]; !ok || (dir.X != 1 && dir.Y != 1) {
		t.Errorf("BuildFlowField failed: wrong direction for (1,1), got %v", dir)
	}
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
	if dir, ok := field[geom.Vec2{1, 0}]; !ok || dir.X != 0 || dir.Y != 1 {
		t.Errorf("BuildFlowField failed: wrong direction for (1,0), got %v", dir)
	}
	if dir, ok := field[geom.Vec2{0, 1}]; !ok || dir.X != 1 || dir.Y != 0 {
		t.Errorf("BuildFlowField failed: wrong direction for (0,1), got %v", dir)
	}
}
