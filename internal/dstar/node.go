package dstar

import (
	"math"

	"github/unng-lab/madfarmer/internal/board"
	"github/unng-lab/madfarmer/internal/geom"
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

var neighbors = [8]geom.Point{
	{0, -1},
	{0, 1},
	{-1, 0},
	{1, 0},
	{-1, -1},
	{-1, 1},
	{1, -1},
	{1, 1},
}

// Node представляет узел в графе.
type Node struct {
	Position  geom.Point // Позиция узла в пространстве.
	G         float64    // Стоимость пути от стартового узла до текущего.
	RHS       float64    // Оценка стоимости от текущего узла до целевого.
	Key       [2]float64 // Ключ узла для очереди с приоритетом.
	Neighbors []*Node    // Соседи текущего узла.
	Obstacle  bool       // Признак наличия препятствия.
}

func (n *Node) Cost(v *Node) float64 {
	if n.Obstacle || v.Obstacle {
		return math.Inf(1)
	}
	// Здесь может быть логика расчета стоимости, например, по-разному для соседей по диагонали и по прямой
	return 1.0
}

func (n Node) heuristic(goalX, goalY float64) float64 {
	return math.Abs(n.Position.X-goalX) + math.Abs(n.Position.Y-goalY)
}

func (n Node) to(target Node) byte {
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

func NewNode(position geom.Point, b *board.Board) *Node {
	return &Node{
		Position:  position,
		G:         0,
		RHS:       0,
		Key:       [2]float64{},
		Neighbors: nil,
		Obstacle:  false,
	}
}
