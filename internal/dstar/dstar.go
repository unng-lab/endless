package dstar

import (
	"errors"
	"math"

	"github.com/unng-lab/madfarmer/internal/board"
	"github.com/unng-lab/madfarmer/internal/geom"
)

const (
	pathCapacity  = 32
	queueCapacity = 512
	smallCapacity = 8
	costsCapacity = 32
	fromsCapacity = 32
)

var errNoPath = errors.New("no Path")

type nodeCacheKey struct {
	x, y int64
}
type DStar struct {
	B           *board.Board
	start, goal *Node
	nodes       []*Node // Очередь с приоритетом (куча)
	km          float64 // Переменная для учета изменений в графе

	nodeCache map[nodeCacheKey]*Node // Cache to store nodes by position
}

func NewDstar(b *board.Board) *DStar {
	return &DStar{
		B:     b,
		start: nil,
		goal:  nil,
		nodes: make([]*Node, 0, queueCapacity),
	}
}

// Инициализация алгоритма D* Lite.
func (ds *DStar) Initialize(startPos, goalPos geom.Point) {
	ds.km = 0
	ds.nodes = []*Node{}
	ds.nodeCache = make(map[nodeCacheKey]*Node)

	ds.start = ds.getNode(startPos)
	ds.goal = ds.getNode(goalPos)

	ds.start.G = math.Inf(1)
	ds.start.RHS = math.Inf(1)

	ds.goal.G = math.Inf(1)
	ds.goal.RHS = 0
	ds.goal.Key = ds.calculateKey(ds.goal)

	ds.Push(ds.goal)
}

// Получение узла по позиции, создание нового при необходимости.
func (ds *DStar) getNode(pos geom.Point) *Node {
	key := nodeCacheKey{int64(pos.X), int64(pos.Y)}
	if node, exists := ds.nodeCache[key]; exists {
		return node
	}
	obstacle := ds.B.IsObstacle(pos)
	node := NewNode(pos, obstacle)
	ds.nodeCache[key] = node
	return node
}

// Проверка корректности позиции (в пределах границ и не препятствие).
func (ds *DStar) isValidPosition(pos geom.Point) bool {
	return ds.B.IsInside(pos) && !ds.B.IsObstacle(pos)
}

// Получение соседей узла.
func (ds *DStar) getNeighbors(u *Node) []*Node {
	neighbors := []*Node{}
	for _, offset := range neighborsOffsets {
		pos := geom.Point{X: u.Position.X + offset.X, Y: u.Position.Y + offset.Y}
		if ds.isValidPosition(pos) {
			neighbors = append(neighbors, ds.getNode(pos))
		}
	}
	u.Neighbors = neighbors
	return neighbors
}

// Вычисление ключа для узла.
func (ds *DStar) calculateKey(n *Node) [2]float64 {
	minGorRHS := math.Min(n.G, n.RHS)
	return [2]float64{
		minGorRHS + n.heuristic(ds.start.Position) + ds.km,
		minGorRHS,
	}
}

// Сравнение ключей узлов.
func compareKeys(a, b [2]float64) int {
	if a[0] < b[0] {
		return -1
	}
	if a[0] > b[0] {
		return 1
	}
	if a[1] < b[1] {
		return -1
	}
	if a[1] > b[1] {
		return 1
	}
	return 0
}

// ComputeShortestPath выполняет вычисление кратчайшего пути.
func (ds *DStar) ComputeShortestPath() {
	for ds.Len() > 0 {
		u := ds.Pop()

		if compareKeys(u.Key, ds.calculateKey(ds.start)) > 0 && ds.start.RHS == ds.start.G {
			break
		}

		if u.G > u.RHS {
			u.G = u.RHS
			for _, s := range ds.getNeighbors(u) {
				ds.UpdateVertex(s)
			}
		} else {
			u.G = math.Inf(1)
			ds.UpdateVertex(u)
			for _, s := range ds.getNeighbors(u) {
				ds.UpdateVertex(s)
			}
		}
	}
}

// Обновление узла в очереди с приоритетом.
func (ds *DStar) UpdateVertex(u *Node) {
	if u.Position != ds.goal.Position {
		u.RHS = math.Inf(1)
		neighbors := ds.getNeighbors(u)
		for _, s := range neighbors {
			u.RHS = min(u.RHS, s.G+s.Cost(u))
		}
	}

	// Если узел уже в открытом списке, удаляем его
	if u.Index >= 0 {
		ds.Remove(u.Index)
	}

	if u.G != u.RHS {
		u.Key = ds.calculateKey(u)
		ds.Push(u)
	}
}

// Метод для получения предшественников узла
func (ds *DStar) predecessors(u *Node) []*Node {
	return u.Neighbors
}

// Метод для получения G-стоимости узла (с проверкой Infinity)
func (ds *DStar) getG(u *Node) float64 {
	if u == nil {
		return math.Inf(1)
	}
	return u.G
}

// Метод стоимости перехода между узлами
func (ds *DStar) Cost(a, b *Node) float64 {
	if a.Obstacle || b.Obstacle {
		return math.Inf(1)
	}
	// If moving diagonally, cost is sqrt(2), else 1
	dx := math.Abs(a.Position.X - b.Position.X)
	dy := math.Abs(a.Position.Y - b.Position.Y)
	if dx+dy > 1 {
		return math.Sqrt(2)
	}
	return 1.0
}
