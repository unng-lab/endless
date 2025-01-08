package main

import (
	"container/heap"
	"fmt"
	"math"
	"strings"
)

// Node представляет узел в сетке
type Node struct {
	x, y       int     // Координаты узла
	g, rhs     float64 // g - текущее значение стоимости, rhs - ожидаемое значение стоимости
	heuristic  float64 // эвристическая оценка до цели
	inOpenList bool    // Флаг, указывающий, находится ли узел в открытом списке
	index      int     // Индекс в приоритетной очереди
}

// PriorityQueue реализует приоритетную очередь для узлов
type PriorityQueue []*Node

// Len возвращает длину очереди
func (pq PriorityQueue) Len() int { return len(pq) }

// Less определяет порядок узлов в очереди
func (pq PriorityQueue) Less(i, j int) bool {
	// Узел с меньшим ключом имеет более высокий приоритет
	// Сравниваем по (min(g, rhs) + heuristic) и затем по min(g, rhs)
	if pq[i].g+pq[i].heuristic == pq[j].g+pq[j].heuristic {
		return pq[i].g < pq[j].g
	}
	return pq[i].g+pq[i].heuristic < pq[j].g+pq[j].heuristic
}

// Swap меняет два элемента местами
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push добавляет элемент в очередь
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	node := x.(*Node)
	node.index = n
	*pq = append(*pq, node)
}

// Pop удаляет и возвращает элемент с наивысшим приоритетом
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	node := old[n-1]
	old[n-1] = nil // Для сборщика мусора
	node.index = -1
	*pq = old[0 : n-1]
	return node
}

// DStarLite реализует алгоритм D* Lite
type DStarLite struct {
	grid          [][]bool      // Сетка препятствий
	width, height int           // Размеры сетки
	start, goal   *Node         // Начальная и конечная точки
	openList      PriorityQueue // Открытый список
}

// NewDStarLite инициализирует новый экземпляр DStarLite
func NewDStarLite(width, height int, startX, startY, goalX, goalY int) *DStarLite {
	// Инициализируем сетку свободных клеток
	grid := make([][]bool, height)
	for i := range grid {
		grid[i] = make([]bool, width)
		for j := range grid[i] {
			grid[i][j] = true // true означает проходимую клетку
		}
	}

	// Создаем начальный и конечный узлы
	start := &Node{x: startX, y: startY, g: math.Inf(1), rhs: math.Inf(1)}
	goal := &Node{x: goalX, y: goalY, g: 0, rhs: 0}

	// Инициализируем открытый список и добавляем цель
	openList := make(PriorityQueue, 0)
	heap.Init(&openList)
	heap.Push(&openList, goal)

	return &DStarLite{
		grid:     grid,
		width:    width,
		height:   height,
		start:    start,
		goal:     goal,
		openList: openList,
	}
}

// heuristic вычисляет эвристическую функцию (эвклидово расстояние)
func heuristic(a, b *Node) float64 {
	return math.Hypot(float64(a.x-b.x), float64(a.y-b.y))
}

// getNeighbors возвращает соседние узлы для данного узла
func (d *DStarLite) getNeighbors(node *Node) []*Node {
	neighbors := []*Node{}
	directions := []struct{ dx, dy int }{
		{-1, 0}, {1, 0}, {0, -1}, {0, 1},

		{-1, -1}, {-1, 1}, {1, -1}, {1, 1},
	}
	for _, dir := range directions {
		nx, ny := node.x+dir.dx, node.y+dir.dy
		if nx >= 0 && ny >= 0 && nx < d.width && ny < d.height && d.grid[ny][nx] {
			neighbor := &Node{x: nx, y: ny, g: 0, rhs: 0}
			neighbors = append(neighbors, neighbor)
		}
	}
	return neighbors
}

// updateVertex обновляет значение узла в системе
func (d *DStarLite) updateVertex(node *Node) {
	if node != d.goal {
		// Обновляем rhs как минимум стоимости через соседей
		minRhs := math.Inf(1)
		for _, neighbor := range d.getNeighbors(node) {
			tentativeRhs := neighbor.g + heuristic(node, neighbor)
			if tentativeRhs < minRhs {
				minRhs = tentativeRhs
			}
		}
		node.rhs = minRhs
	}

	// Если узел уже в открытом списке, удаляем его
	// Здесь можно реализовать удаление узла из открытого списка, если он там есть

	// Если g != rhs, то добавляем узел в открытый список
	if node.g != node.rhs {
		heap.Push(&d.openList, node)
		node.inOpenList = true
	}
}

// computeShortestPath вычисляет кратчайший путь
func (d *DStarLite) computeShortestPath() {
	for d.openList.Len() > 0 {
		current := heap.Pop(&d.openList).(*Node)
		current.inOpenList = false

		if current.g > current.rhs {
			current.g = current.rhs
			// Проходимся по всем соседям и обновляем их вершины
			for _, neighbor := range d.getNeighbors(current) {
				d.updateVertex(neighbor)
			}
		} else {
			current.g = math.Inf(1)
			d.updateVertex(current)
			for _, neighbor := range d.getNeighbors(current) {
				d.updateVertex(neighbor)
			}
		}

		// Проверяем, достигли ли мы стартовой точки
		if current == d.start && d.start.g == d.start.rhs {
			break
		}
	}
}

// reconstructPath восстанавливает путь от старта до цели
func (d *DStarLite) reconstructPath() []*Node {
	path := []*Node{}
	current := d.start
	if current.g == math.Inf(1) {
		// Нет доступного пути
		return path
	}
	path = append(path, current)
	for current != d.goal {
		minCost := math.Inf(1)
		var next *Node
		for _, neighbor := range d.getNeighbors(current) {
			cost := neighbor.g + heuristic(current, neighbor)
			if cost < minCost {
				minCost = cost
				next = neighbor
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

// updateGrid обновляет сетку и пересчитывает пути при изменениях
func (d *DStarLite) updateGrid(x, y int, walkable bool) {
	d.grid[y][x] = walkable
	// Нужно обновить узлы и пересчитать пути
	node := &Node{x: x, y: y}
	d.updateVertex(node)
	d.computeShortestPath()
}

// display отображает сетку и путь в консоли
func (d *DStarLite) display(path []*Node) {
	gridDisplay := make([][]string, d.height)
	for i := range gridDisplay {
		gridDisplay[i] = make([]string, d.width)
		for j := range gridDisplay[i] {
			if !d.grid[i][j] {
				gridDisplay[i][j] = "█" // Препятствие
			} else {
				gridDisplay[i][j] = " " // Проходимая клетка
			}
		}
	}

	// Помечаем путь
	for _, node := range path {
		if node != d.start && node != d.goal {
			gridDisplay[node.y][node.x] = "*" // Часть пути
		}
	}

	// Отмечаем старт и цель
	gridDisplay[d.start.y][d.start.x] = "S" // Старт
	gridDisplay[d.goal.y][d.goal.x] = "G"   // Цель

	// Выводим сетку
	var sb strings.Builder
	for _, row := range gridDisplay {
		sb.WriteString(strings.Join(row, " "))
		sb.WriteString("\n")
	}
	fmt.Println(sb.String())
}

func main() {
	// Размеры сетки
	width, height := 10, 10

	// Координаты стартовой и конечной точек

	startX, startY := 0, 0
	goalX, goalY := 9, 9

	// Инициализируем алгоритм
	dstar := NewDStarLite(width, height, startX, startY, goalX, goalY)

	// Первоначальный расчет пути
	dstar.computeShortestPath()
	path := dstar.reconstructPath()
	fmt.Println("Первоначальный путь:")
	dstar.display(path)

	// Вносим изменения в сетку (например, добавляем препятствия)
	fmt.Println("Добавляем препятствия и обновляем путь...")
	obstacles := []struct{ x, y int }{
		{5, 0}, {5, 1}, {5, 2}, {5, 3}, {5, 4},
		{5, 5}, {5, 6}, {5, 7}, {5, 8}, {5, 9},
	}
	for _, obs := range obstacles {
		dstar.updateGrid(obs.x, obs.y, false)
	}

	// Пересчитываем путь после изменений
	path = dstar.reconstructPath()
	fmt.Println("Путь после добавления препятствий:")
	dstar.display(path)
}
