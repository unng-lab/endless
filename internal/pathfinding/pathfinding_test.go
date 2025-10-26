// pathfinding/pathfinding_test.go
package pathfinding

import (
	"testing"

	"github.com/unng-lab/endless/internal/geom"
	"github.com/unng-lab/endless/internal/navmesh"
)

func TestStringPulling(t *testing.T) {
	// Создаем простые порталы
	portals := [][2]geom.Vec2{
		{{1, 1}, {3, 1}},
		{{2, 2}, {4, 2}},
		{{3, 3}, {5, 3}},
	}

	start := geom.Vec2{0, 0}
	goal := geom.Vec2{6, 6}

	path := StringPulling(portals, start, goal)

	if len(path) != 5 {
		t.Errorf("StringPulling failed: expected 5 points, got %d", len(path))
	}

	// Проверяем начальную и конечную точки
	if !path[0].Eq(start) {
		t.Errorf("StringPulling failed: start point mismatch, got %v", path[0])
	}
	if !path[len(path)-1].Eq(goal) {
		t.Errorf("StringPulling failed: goal point mismatch, got %v", path[len(path)-1])
	}

	// Проверяем, что промежуточные точки находятся внутри порталов
	for i := 1; i < len(path)-1; i++ {
		p := path[i]
		portal := portals[i-1]

		minX := geom.Min(portal[0].X, portal[1].X)
		maxX := geom.Max(portal[0].X, portal[1].X)
		minY := geom.Min(portal[0].Y, portal[1].Y)
		maxY := geom.Max(portal[0].Y, portal[1].Y)

		if p.X < minX || p.X > maxX || p.Y < minY || p.Y > maxY {
			t.Errorf("StringPulling failed: point %v outside portal %v-%v", p, portal[0], portal[1])
		}
	}
}

func TestBuildPortalsFromPolySequence(t *testing.T) {
	// Создаем простую последовательность прямоугольников
	polys := []*navmesh.RectPoly{
		{Min: geom.Vec2{0, 0}, Max: geom.Vec2{5, 5}, Centroid: geom.Vec2{2, 2}},
		{Min: geom.Vec2{5, 0}, Max: geom.Vec2{10, 5}, Centroid: geom.Vec2{7, 2}},
		{Min: geom.Vec2{5, 5}, Max: geom.Vec2{10, 10}, Centroid: geom.Vec2{7, 7}},
	}

	portals := BuildPortalsFromPolySequence(polys)

	if len(portals) != 2 {
		t.Errorf("BuildPortalsFromPolySequence failed: expected 2 portals, got %d", len(portals))
	}

	// Проверяем первый портал (между первым и вторым полигоном)
	if portals[0][0].X != 5 || portals[0][0].Y != 0 || portals[0][1].X != 5 || portals[0][1].Y != 5 {
		t.Errorf("BuildPortalsFromPolySequence failed: first portal mismatch, got %v-%v", portals[0][0], portals[0][1])
	}

	// Проверяем второй портал (между вторым и третьим полигоном)
	if portals[1][0].X != 5 || portals[1][0].Y != 5 || portals[1][1].X != 10 || portals[1][1].Y != 5 {
		t.Errorf("BuildPortalsFromPolySequence failed: second portal mismatch, got %v-%v", portals[1][0], portals[1][1])
	}
}

func TestHeuristicPoly(t *testing.T) {
	// Создаем два узла
	node1 := &polyNode{poly: &navmesh.RectPoly{Centroid: geom.Vec2{0, 0}}}
	node2 := &polyNode{poly: &navmesh.RectPoly{Centroid: geom.Vec2{3, 4}}}

	heuristic := HeuristicPoly(node1, node2)
	if heuristic != 7 { // |3-0| + |4-0| = 3 + 4 = 7
		t.Errorf("HeuristicPoly failed: expected 7, got %d", heuristic)
	}
}
