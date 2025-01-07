package dstarlite

import (
	"container/heap"
	"fmt"
	"math"
)

type Point struct {
	X, Y int
}

type Node struct {
	Pos       Point
	G, Rhs    float64
	Key       [2]float64
	Obstacle  bool
	Neighbors []*Node
}

type Grid struct {
	Cells       [][]*Node
	OpenList    PriorityQueue
	Start, Goal *Node
}

func NewGrid(width, height int, start, goal Point) *Grid {
	cells := make([][]*Node, height)
	for y := 0; y < height; y++ {
		cells[y] = make([]*Node, width)
		for x := 0; x < width; x++ {
			cell := &Node{
				Pos: Point{X: x, Y: y},
				G:   math.Inf(1),
				Rhs: math.Inf(1),
				Key: [2]float64{math.Inf(1), math.Inf(1)},
			}
			cells[y][x] = cell
		}
	}
	grid := &Grid{
		Cells: cells,
		Start: cells[start.Y][start.X],
		Goal:  cells[goal.Y][goal.X],
	}
	grid.Initialize()
	return grid
}

func (grid *Grid) Initialize() {
	grid.Goal.Rhs = 0
	grid.UpdateKey(grid.Goal)
	heap.Init(&grid.OpenList)
	heap.Push(&grid.OpenList, grid.Goal)
}

func (grid *Grid) UpdateVertex(u *Node) {
	if u != grid.Goal {
		minRhs := math.Inf(1)
		for _, s := range u.Neighbors {
			if cost := s.G + grid.Cost(u, s); cost < minRhs {
				minRhs = cost
			}
		}
		u.Rhs = minRhs
	}
	grid.RemoveFromOpenList(u)
	if u.G != u.Rhs {
		grid.UpdateKey(u)
		heap.Push(&grid.OpenList, u)
	}
}

func (grid *Grid) ComputeShortestPath() {
	i := 0
	for len(grid.OpenList) > 0 && (grid.OpenList[0].Key[0] < grid.CalculateKey(grid.Start)[0] ||
		(grid.OpenList[0].Key[0] == grid.CalculateKey(grid.Start)[0] && grid.OpenList[0].Key[1] < grid.CalculateKey(grid.Start)[1]) ||
		grid.Start.Rhs != grid.Start.G) {
		i++
		u := heap.Pop(&grid.OpenList).(*Node)
		if u.G > u.Rhs {
			u.G = u.Rhs
			for _, s := range u.Neighbors {
				grid.UpdateVertex(s)
			}
		} else {
			u.G = math.Inf(1)
			grid.UpdateVertex(u)
			for _, s := range u.Neighbors {
				grid.UpdateVertex(s)
			}
		}
	}
	println("\nИтераций", i)
}

func (grid *Grid) Cost(u, v *Node) float64 {
	if u.Obstacle || v.Obstacle {
		return math.Inf(1)
	}
	// Здесь может быть логика расчета стоимости, например, по-разному для соседей по диагонали и по прямой
	return 1.0
}

func (u *Node) SetCostTo(v *Node, cost float64) {
	// Здесь нужно обновить структуру, хранящую стоимость перехода
	// Предположим, что у нас есть карта стоимостей
	if cost == math.Inf(1) {
		v.Obstacle = true
	} else {
		v.Obstacle = false
	}
}

func (u *Node) Successors() []*Node {
	var successors []*Node
	for _, v := range u.Neighbors {
		if !v.Obstacle {
			successors = append(successors, v)
		}
	}
	return successors
}

func (grid *Grid) UpdateEdge(u, v *Node, cost float64) {
	oldCost := grid.Cost(u, v)
	u.SetCostTo(v, cost)

	// Если стоимость увеличилась
	if oldCost < cost {
		if u.Rhs == oldCost+v.G {
			// Пересчитываем rhs(u)
			if u != grid.Goal {
				u.Rhs = math.Inf(1)
				for _, s := range u.Successors() {
					u.Rhs = math.Min(u.Rhs, grid.Cost(u, s)+s.G)
				}
			}
			grid.UpdateVertex(u)
		}
	} else {
		// Если стоимость уменьшилась
		if u != grid.Goal {
			u.Rhs = math.Min(u.Rhs, grid.Cost(u, v)+v.G)
		}
		grid.UpdateVertex(u)
	}
}

func (grid *Grid) RemoveFromOpenList(u *Node) {
	for i, cell := range grid.OpenList {
		if cell == u {
			heap.Remove(&grid.OpenList, i)
			break
		}
	}
}

func (grid *Grid) UpdateKey(u *Node) {
	u.Key = grid.CalculateKey(u)
}

func (grid *Grid) CalculateKey(u *Node) [2]float64 {
	k := [2]float64{
		math.Min(u.G, u.Rhs) + grid.Heuristic(grid.Start, u),
		math.Min(u.G, u.Rhs),
	}
	return k
}

func (grid *Grid) Heuristic(a, b *Node) float64 {
	dx := math.Abs(float64(a.Pos.X - b.Pos.X))
	dy := math.Abs(float64(a.Pos.Y - b.Pos.Y))
	return dx + dy
}

type PriorityQueue []*Node

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].Key[0] == pq[j].Key[0] {
		return pq[i].Key[1] < pq[j].Key[1]
	}
	return pq[i].Key[0] < pq[j].Key[0]
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*Node)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	n := len(*pq)
	item := (*pq)[n-1]
	*pq = (*pq)[0 : n-1]
	return item
}

func ExtractPath(grid *Grid) []*Node {
	path := []*Node{}
	current := grid.Start
	for current != grid.Goal {
		minCost := math.Inf(1)
		var next *Node
		for _, s := range current.Neighbors {
			cost := grid.Cost(current, s) + s.G
			if cost < minCost {
				minCost = cost
				next = s
			}
		}
		if next == nil {
			break
		}
		path = append(path, next)
		current = next
	}
	return path
}

func PrintPath(path []*Node) {
	for _, cell := range path {
		fmt.Printf("(%d, %d) ", cell.Pos.X, cell.Pos.Y)
	}
	fmt.Println()
}

func (grid *Grid) PrintGrid() {
	for y := 0; y < len(grid.Cells); y++ {
		for x := 0; x < len(grid.Cells[0]); x++ {
			cell := grid.Cells[y][x]
			switch {
			case cell == grid.Start:
				fmt.Print("S")
			case cell == grid.Goal:
				fmt.Print("G")
			case cell.Obstacle:
				fmt.Print("#")
			default:
				fmt.Print(".")
			}
		}
		fmt.Println()
	}
}

func (grid *Grid) PrintPathOnGrid(path []*Node) {
	pathMap := make(map[Point]bool)
	for _, cell := range path {
		pathMap[cell.Pos] = true
	}

	for y := 0; y < len(grid.Cells); y++ {
		for x := 0; x < len(grid.Cells[0]); x++ {
			cell := grid.Cells[y][x]
			pos := cell.Pos
			switch {
			case cell == grid.Start:
				fmt.Print("S")
			case cell == grid.Goal:
				fmt.Print("G")
			case pathMap[pos]:
				fmt.Print("*")
			case cell.Obstacle:
				fmt.Print("#")
			default:
				fmt.Print(".")
			}
		}
		fmt.Println()
	}
}

type EdgeUpdate struct {
	U, V *Node
	Cost float64
}

func (grid *Grid) UpdateEdges(updates ...EdgeUpdate) {
	for _, update := range updates {
		u := update.U
		v := update.V
		cost := update.Cost

		grid.UpdateEdge(u, v, cost)
	}
	grid.ComputeShortestPath()
}

// Функция для получения соседей клетки в координатах (x, y)
func getNeighbors(x, y int, grid [][]*Node, width, height int) []*Node {
	neighbors := []*Node{}

	// Смещения для соседних клеток (по восьми направлениям)
	directions := [][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	for _, dir := range directions {
		nx, ny := x+dir[0], y+dir[1]
		// Проверяем, что новые координаты внутри границ карты
		if nx >= 0 && nx < width && ny >= 0 && ny < height {
			neighbor := grid[ny][nx]
			neighbors = append(neighbors, neighbor)
		}
	}

	return neighbors
}
