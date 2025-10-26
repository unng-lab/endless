// pathfinding/pathfinding.go
package pathfinding

import (
	"math"

	"github.com/unng-lab/endless/internal/chunk"
	"github.com/unng-lab/endless/internal/geom"
	"github.com/unng-lab/endless/internal/navmesh"
)

// FindPathOnNavMesh находит путь в пределах локальных NavMesh
func FindPathOnNavMesh(cm *chunk.ChunkManager, start, goal geom.Vec2) ([]geom.Vec2, error) {
	// Находим чанки и NavMesh для start и goal
	cs, _ := geom.WorldToChunk(start, chunk.ChunkSize)
	ce, _ := geom.WorldToChunk(goal, chunk.ChunkSize)
	_ = cm.EnsureLoaded(cs)
	_ = cm.EnsureLoaded(ce)

	// A* на графе полигонов
	// (упрощенная реализация для демонстрации)

	// Для примера вернем прямой путь
	return []geom.Vec2{start, goal}, nil
}

// StringPulling реализует алгоритм funnel для сглаживания пути
func StringPulling(portals [][2]geom.Vec2, start, goal geom.Vec2) []geom.Vec2 {
	// Упрощенная реализация для демонстрации
	path := []geom.Vec2{start}
	for _, p := range portals {
		// Берем середину каждого портала
		mid := geom.Vec2{(p[0].X + p[1].X) / 2, (p[0].Y + p[1].Y) / 2}
		path = append(path, mid)
	}
	path = append(path, goal)
	return path
}

// BuildPortalsFromPolySequence строит порталы из последовательности полигонов
func BuildPortalsFromPolySequence(polys []*navmesh.RectPoly) [][2]geom.Vec2 {
	if len(polys) < 2 {
		return nil
	}

	portals := make([][2]geom.Vec2, 0, len(polys)-1)
	for i := 0; i < len(polys)-1; i++ {
		a := polys[i]
		b := polys[i+1]

		minx := geom.Max(a.Min.X, b.Min.X)
		maxx := geom.Min(a.Max.X, b.Max.X)
		miny := geom.Max(a.Min.Y, b.Min.Y)
		maxy := geom.Min(a.Max.Y, b.Max.Y)

		if minx > maxx || miny > maxy {
			portals = append(portals, [2]geom.Vec2{a.Centroid, b.Centroid})
			continue
		}

		switch {
		case minx == maxx && miny == maxy:
			p := geom.Vec2{minx, miny}
			portals = append(portals, [2]geom.Vec2{p, p})
		case minx == maxx:
			x := minx
			portals = append(portals, [2]geom.Vec2{
				{x, miny},
				{x, maxy},
			})
		case miny == maxy:
			y := miny
			portals = append(portals, [2]geom.Vec2{
				{minx, y},
				{maxx, y},
			})
		default:
			midX := (minx + maxx) / 2
			portals = append(portals, [2]geom.Vec2{
				{midX, miny},
				{midX, maxy},
			})
		}
	}
	return portals
}

// polyNode — узел для A* на уровне полигонов
type polyNode struct {
	poly *navmesh.RectPoly
	g, f int
	neis []*polyNode
}

// polyPQ — очередь с приоритетом для A*
type polyPQ []*polyNode

func (pq polyPQ) Len() int           { return len(pq) }
func (pq polyPQ) Less(i, j int) bool { return pq[i].f < pq[j].f }
func (pq polyPQ) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *polyPQ) Push(x interface{}) {
	*pq = append(*pq, x.(*polyNode))
}

func (pq *polyPQ) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[0 : n-1]
	return x
}

// HeuristicPoly — эвристика для узла полигона
func HeuristicPoly(a, b *polyNode) int {
	dx := a.poly.Centroid.X - b.poly.Centroid.X
	dy := a.poly.Centroid.Y - b.poly.Centroid.Y
	return int(math.Abs(float64(dx)) + math.Abs(float64(dy)))
}
