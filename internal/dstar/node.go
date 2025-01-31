package dstar

import (
	"math"
	"sync"

	"github.com/unng-lab/madfarmer/internal/board"
	"github.com/unng-lab/madfarmer/internal/geom"
)

const (
	DirNone byte = iota
	DirUp        = iota
	DirUpRight
	DirRight
	DirDownRight
	DirDown
	DirDownLeft
	DirLeft
	DirUpLeft
)

var neighborsOffsets = [8]geom.Point{
	{0, -1},
	{0, 1},
	{-1, 0},
	{1, 0},
	{-1, -1},
	{-1, 1},
	{1, -1},
	{1, 1},
}

var nodePool = sync.Pool{
	New: func() interface{} {
		return &Node{
			Neighbors: make([]*Node, 0, 8),
		}
	},
}

// Node представляет узел в графе.
type Node struct {
	Position  geom.Point // Позиция узла в пространстве.
	G         float64    // Стоимость пути от стартового узла до текущего.
	RHS       float64    // Оценка стоимости от текущего узла до целевого.
	Key       [2]float64 // Ключ узла для очереди с приоритетом.
	Neighbors []*Node    // Соседи текущего узла.
	Index     int
}

// Стоимость перехода между узлами.
func (n *Node) Cost(v *Node, b *board.Board) float64 {
	if n.Position.To(v.Position) == geom.DirNone {
		return math.Inf(1)
	}
	cost := b.GetCost(n.Position, v.Position, 0)
	return cost
}

// Эвристическая функция — манхэттенское расстояние.
func (n *Node) heuristic(goal geom.Point) float64 {
	// Using Euclidean distance as heuristic
	dx := n.Position.X - goal.X
	dy := n.Position.Y - goal.Y
	return math.Hypot(dx, dy)
}

func (n *Node) to(target Node) byte {
	switch x, y := target.Position.X-n.Position.X, target.Position.Y-n.Position.Y; {
	case x == 0 && y == -1:
		return DirUp
	case x == 0 && y == 1:
		return DirDown
	case x == -1 && y == 0:
		return DirLeft
	case x == 1 && y == 0:
		return DirRight
	case x == -1 && y == -1:
		return DirUpLeft
	case x == -1 && y == 1:
		return DirDownLeft
	case x == 1 && y == -1:
		return DirUpRight
	case x == 1 && y == 1:
		return DirDownRight
	default:
		return DirNone
	}
}

// NewNode создаёт новый узел.
func NewNode(position geom.Point) *Node {
	newNode := nodePool.Get().(*Node)
	newNode.reset()
	newNode.Position = position
	return newNode
}

func (n *Node) reset() {
	n.G = math.Inf(1)
	n.RHS = math.Inf(1)
	n.Neighbors = n.Neighbors[:0]
	n.Index = -1
}
