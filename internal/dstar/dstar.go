package dstar

import (
	"math"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/geom"
)

const (
	pathCapacity  = 32
	queueCapacity = 512
	smallCapacity = 8
	costsCapacity = 32
	fromsCapacity = 32
)

type Dstar struct {
	B           *board.Board
	start, goal *Node
	nodes       []*Node
	Path        []geom.Point
}

func NewDstar(b *board.Board) Dstar {
	return Dstar{
		B:     b,
		nodes: make([]*Node, 0, queueCapacity),
		Path:  make([]geom.Point, 0, pathCapacity),
	}
}

func (ds *Dstar) update(node *Node, key [2]float64) {
	node.Key = key
	ds.Fix(node.Index)
}

// heuristic оценивает стоимость от узла до цели.
func heuristic(a, b geom.Point) float64 {
	return math.Abs(a.X-b.X) + math.Abs(a.Y-b.Y) // Манхэттенское расстояние.
}

// computeKey вычисляет ключ для узла.
func computeKey(node *Node, start *Node, km float64) [2]float64 {
	min := math.Min(node.G, node.RHS)
	return [2]float64{min + heuristic(node.Position, start.Position) + km, min}
}

// updateNode обновляет информацию об узле и его позицию в очереди.
func (ds *Dstar) updateNode(node *Node, goal *Node, km float64) {
	if node != goal {
		minRHS := math.Inf(1)
		for _, succ := range node.Neighbors {
			cost := 1.0 // Предполагаем, что стоимость перехода равна 1.
			if succ.G+cost < minRHS {
				minRHS = succ.G + cost
			}
		}
		node.RHS = minRHS
	}

	index := node.Index
	if node.G != node.RHS {
		key := computeKey(node, goal, km)
		if node.InQueue {
			ds.update(node, key)
		} else {
			node.Key = key
			ds.Push(node)
			node.InQueue = true
		}
	} else if node.InQueue {
		ds.Remove(index)
		node.InQueue = false
	}
}

// computeShortestPath вычисляет кратчайший путь, обновляя только необходимые узлы.
// Это основное отличие от A*, где всегда пересчитывается путь с нуля.
func (ds *Dstar) computeShortestPath(start *Node, goal *Node, km *float64) {
	for ds.Len() > 0 && (less((ds.nodes)[0].Key, computeKey(start, goal, *km)) || start.RHS != start.G) {
		node := ds.Pop()
		node.InQueue = false

		if node.G > node.RHS {
			node.G = node.RHS
			for _, pred := range node.Neighbors {
				ds.updateNode(pred, goal, *km)
			}
		} else {
			node.G = math.Inf(1)
			ds.updateNode(node, goal, *km)
			for _, pred := range node.Neighbors {
				ds.updateNode(pred, goal, *km)
			}
		}
	}
}

// less сравнивает два ключа.
func less(a, b [2]float64) bool {
	if a[0] == b[0] {
		return a[1] < b[1]
	}
	return a[0] < b[0]
}

// Изменение узла (например, появление препятствия).
func (ds *Dstar) updateEdge(node *Node, km *float64, goal *Node) {
	// Здесь мы моделируем изменение, например, добавление препятствия.
	// В этом случае мы можем изменить соседей узла или стоимость перехода.
	node.RHS = math.Inf(1)
	ds.updateNode(node, goal, *km)
}
