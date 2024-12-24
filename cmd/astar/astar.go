package main

import (
	"container/heap"
	"fmt"
	"math"
)

// Coordinate представляет собой позицию в сетке.
type Coordinate struct {
	X, Y int
}

// Node представляет каждую ячейку в сетке.
type Node struct {
	Coord      Coordinate
	G          float64 // Фактическая стоимость достижения этого узла от начального узла.
	RHS        float64 // Значение одношагового прогноза.
	Key        [2]float64
	IsObstacle bool // true, если узел является непреодолимым препятствием.
}

// NodeHeap реализует интерфейс heap.Interface и содержит узлы Node.
type NodeHeap []*Node

func (h NodeHeap) Len() int { return len(h) }

func (h NodeHeap) Less(i, j int) bool {
	// Узлы упорядочены по значению ключа.
	if h[i].Key[0] == h[j].Key[0] {
		return h[i].Key[1] < h[j].Key[1]
	}
	return h[i].Key[0] < h[j].Key[0]
}

func (h NodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *NodeHeap) Push(x interface{}) {
	*h = append(*h, x.(*Node))
}

func (h *NodeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	node := old[n-1]
	*h = old[0 : n-1]
	return node
}

// DLite представляет алгоритм D* Lite.
type DLite struct {
	Grid         [][]*Node
	Start, Goal  *Node
	U            NodeHeap
	km           float64
	visited      map[*Node]bool
	Last         *Node
	predecessors map[*Node][]*Node
	successors   map[*Node][]*Node
}

// NewDLite инициализирует алгоритм D* Lite.
func NewDLite(grid [][]*Node, startCoord, goalCoord Coordinate) *DLite {
	dlite := &DLite{
		Grid:         grid,
		U:            make(NodeHeap, 0),
		km:           0,
		visited:      make(map[*Node]bool),
		predecessors: make(map[*Node][]*Node),
		successors:   make(map[*Node][]*Node),
	}

	heap.Init(&dlite.U)

	// Инициализация стартовых и целевых узлов.
	dlite.Start = dlite.Grid[startCoord.X][startCoord.Y]
	dlite.Goal = dlite.Grid[goalCoord.X][goalCoord.Y]
	dlite.Last = dlite.Start

	// Инициализация значений RHS и G для всех узлов.
	for i := range dlite.Grid {
		for j := range dlite.Grid[0] {
			node := dlite.Grid[i][j]
			node.G = math.Inf(1)
			node.RHS = math.Inf(1)
			node.Key = [2]float64{math.Inf(1), math.Inf(1)}
		}
	}

	dlite.Goal.RHS = 0
	dlite.Goal.Key = dlite.CalculateKey(dlite.Goal)
	heap.Push(&dlite.U, dlite.Goal)

	return dlite
}

// CalculateKey вычисляет ключ для узла.
func (dlite *DLite) CalculateKey(u *Node) [2]float64 {
	minG_RHS := math.Min(u.G, u.RHS)
	return [2]float64{
		minG_RHS + dlite.Heuristic(dlite.Start, u) + dlite.km,
		minG_RHS,
	}
}

// Heuristic оценивает стоимость от узла a до узла b.
func (dlite *DLite) Heuristic(a, b *Node) float64 {
	// Используется манхэттенское расстояние в качестве эвристики.
	return math.Abs(float64(a.Coord.X-b.Coord.X)) + math.Abs(float64(a.Coord.Y-b.Coord.Y))
}

// UpdateVertex обновляет узел в приоритетной очереди.
func (dlite *DLite) UpdateVertex(u *Node) {
	if u != dlite.Goal {
		u.RHS = dlite.MinSuccessor(u)
	}
	dlite.RemoveFromHeap(u)
	if u.G != u.RHS {
		u.Key = dlite.CalculateKey(u)
		heap.Push(&dlite.U, u)
	}
}

// MinSuccessor находит минимальное значение RHS среди преемников.
func (dlite *DLite) MinSuccessor(u *Node) float64 {
	min := math.Inf(1)
	for _, s := range dlite.GetSuccessors(u) {
		cost := dlite.Cost(u, s)
		if cost+dlite.G(s) < min {
			min = cost + dlite.G(s)
		}
	}
	return min
}

// Cost возвращает стоимость перехода от u к v.
func (dlite *DLite) Cost(u, v *Node) float64 {
	if v.IsObstacle {
		return math.Inf(1)
	}
	return 1 // Предполагается равная стоимость перехода.
}

// G возвращает значение G для узла.
func (dlite *DLite) G(u *Node) float64 {
	return u.G
}

// ComputeShortestPath вычисляет кратчайший путь на основе текущей информации.

func (dlite *DLite) ComputeShortestPath() {
	for dlite.U.Len() > 0 && (dlite.CompareKeys(dlite.U[0].Key, dlite.CalculateKey(dlite.Start)) || dlite.Start.RHS != dlite.Start.G) {
		u := heap.Pop(&dlite.U).(*Node)
		if u.G > u.RHS {
			u.G = u.RHS
			for _, s := range dlite.GetPredecessors(u) {
				dlite.UpdateVertex(s)
			}
		} else {
			u.G = math.Inf(1)
			dlite.UpdateVertex(u)
			for _, s := range dlite.GetPredecessors(u) {
				dlite.UpdateVertex(s)
			}
		}
	}
}

// CompareKeys сравнивает два ключа.
func (dlite *DLite) CompareKeys(k1, k2 [2]float64) bool {
	if k1[0] < k2[0] {
		return true
	}
	if k1[0] > k2[0] {
		return false
	}
	return k1[1] < k2[1]
}

// GetSuccessors возвращает преемников узла.
func (dlite *DLite) GetSuccessors(u *Node) []*Node {
	if succ, ok := dlite.successors[u]; ok {
		return succ
	}
	succ := dlite.Neighbors(u)
	dlite.successors[u] = succ
	return succ
}

// GetPredecessors возвращает предшественников узла.
func (dlite *DLite) GetPredecessors(u *Node) []*Node {
	if pred, ok := dlite.predecessors[u]; ok {
		return pred
	}
	pred := dlite.Neighbors(u)
	dlite.predecessors[u] = pred
	return pred
}

// Neighbors возвращает соседние узлы, которые можно пройти.
func (dlite *DLite) Neighbors(u *Node) []*Node {
	var neighbors []*Node
	dx := []int{-1, 1, 0, 0}
	dy := []int{0, 0, -1, 1}
	for i := 0; i < 4; i++ {
		x := u.Coord.X + dx[i]
		y := u.Coord.Y + dy[i]
		if x >= 0 && x < len(dlite.Grid) && y >= 0 && y < len(dlite.Grid[0]) {
			v := dlite.Grid[x][y]
			neighbors = append(neighbors, v)
		}
	}
	return neighbors
}

// RemoveFromHeap удаляет узел из приоритетной очереди, если он там есть.
func (dlite *DLite) RemoveFromHeap(u *Node) {
	for i, node := range dlite.U {
		if node == u {
			heap.Remove(&dlite.U, i)
			break
		}
	}
}

// Replan обновляет план при изменении препятствий.
func (dlite *DLite) Replan(changedEdges [][2]Coordinate) {
	dlite.km += dlite.Heuristic(dlite.Last, dlite.Start)
	dlite.Last = dlite.Start

	for _, edge := range changedEdges {
		uCoord, vCoord := edge[0], edge[1]
		u := dlite.Grid[uCoord.X][uCoord.Y]
		v := dlite.Grid[vCoord.X][vCoord.Y]
		dlite.UpdateVertex(u)
		dlite.UpdateVertex(v)
	}

	dlite.ComputeShortestPath()
}

// FindPath возвращает путь от стартового узла к целевому узлу.
func (dlite *DLite) FindPath() []Coordinate {
	path := []Coordinate{dlite.Start.Coord}
	current := dlite.Start
	for current != dlite.Goal {
		if current.RHS == math.Inf(1) {
			return nil // Путь не найден.
		}
		minRHS := math.Inf(1)
		var next *Node
		for _, s := range dlite.GetSuccessors(current) {
			if dlite.Cost(current, s)+s.G < minRHS {
				minRHS = dlite.Cost(current, s) + s.G
				next = s
			}
		}
		if next == nil {
			return nil // Путь не найден.
		}
		path = append(path, next.Coord)
		current = next
	}
	return path
}

// Главная функция для демонстрации алгоритма D* Lite.
func main() {
	// Создаем сетку.
	width, height := 10, 10
	grid := make([][]*Node, width)
	for i := range grid {
		grid[i] = make([]*Node, height)
		for j := range grid[i] {
			grid[i][j] = &Node{
				Coord:      Coordinate{X: i, Y: j},
				IsObstacle: false,
			}
		}
	}

	// Определяем стартовые и целевые координаты.
	startCoord := Coordinate{X: 0, Y: 0}
	goalCoord := Coordinate{X: 9, Y: 9}

	// Добавляем некоторые препятствия.
	grid[2][2].IsObstacle = true
	grid[2][3].IsObstacle = true
	grid[2][4].IsObstacle = true
	grid[2][5].IsObstacle = true

	// Инициализируем D* Lite.
	dlite := NewDLite(grid, startCoord, goalCoord)

	// Вычисляем кратчайший путь.
	dlite.ComputeShortestPath()

	// Находим путь.
	path := dlite.FindPath()

	// Выводим путь.
	if path != nil {
		fmt.Println("Найден путь:")
		for _, coord := range path {
			fmt.Printf("(%d, %d) ", coord.X, coord.Y)
		}
		fmt.Println()
	} else {
		fmt.Println("Путь не найден.")
	}
}
