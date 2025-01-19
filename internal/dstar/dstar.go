package dstar

import (
	"errors"
	"fmt"
	"log/slog"
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

func NewDStar(b *board.Board) *DStar {
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

func (d *DStar) MoveStart(newStart *Node) {
	oldStart := d.start
	d.km += oldStart.heuristic(newStart.Position)
	d.start = newStart
	d.UpdateVertex(oldStart)
	d.UpdateVertex(d.start)
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

// Сравнение ключей узлов возвращает true, если первый больше
func compareKeys(a, b [2]float64) bool {
	if a[0] == b[0] {
		return a[1] > b[1]
	}
	return a[0] > b[0]
}

// ComputeShortestPath выполняет вычисление кратчайшего пути. Возвращает количество шагов и ошибку если путь не существует.
func (ds *DStar) ComputeShortestPath() (int, error) {
	i := 0
	for ds.Len() > 0 {
		i++
		slog.Info("Current", "step", i)
		u := ds.Pop()

		if compareKeys(u.Key, ds.calculateKey(ds.start)) && ds.start.RHS == ds.start.G {
			//panic("Shortest path not found")
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
		i = i + 0
		slog.Info("current", "node ", u)
		i = i - 0
	}
	slog.Debug("Shortest path computed in ", " steps ", i)
	return i, nil
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

// Вспомогательная функция для восстановления пути.
func reconstructPath(ds *DStar) ([]geom.Point, error) {
	path := []geom.Point{}
	current := ds.start
	for {
		path = append(path, current.Position)
		if current.Position == ds.goal.Position {
			break
		}
		if current.G == math.Inf(1) {
			return nil, fmt.Errorf("путь не найден")
		}
		minCost := math.Inf(1)
		var nextNode *Node
		for _, neighbor := range ds.getNeighbors(current) {
			cost := current.Cost(neighbor) + neighbor.G
			if cost < minCost {
				minCost = cost
				nextNode = neighbor
			}
		}
		if nextNode == nil {
			return nil, fmt.Errorf("путь оборвался")
		}
		current = nextNode
	}
	return path, nil
}
