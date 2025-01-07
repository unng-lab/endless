package dstarlite

import (
	"fmt"
	"math"
	"testing"
)

func TestPathChangesWithEdgeUpdates(t *testing.T) {
	start := Point{X: 0, Y: 0}
	goal := Point{X: 4, Y: 4}
	grid := NewGrid(5, 5, start, goal)
	fmt.Println("Первоначальная сетка:")
	grid.PrintGrid()
	// Инициализируем соседей для каждой клетки
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			cell := grid.Cells[y][x]
			neighbors := []*Node{}
			if x > 0 {
				neighbors = append(neighbors, grid.Cells[y][x-1])
			}
			if x < 4 {
				neighbors = append(neighbors, grid.Cells[y][x+1])
			}
			if y > 0 {
				neighbors = append(neighbors, grid.Cells[y-1][x])
			}
			if y < 4 {
				neighbors = append(neighbors, grid.Cells[y+1][x])
			}
			cell.Neighbors = neighbors
		}
	}

	grid.ComputeShortestPath()

	path := ExtractPath(grid)
	fmt.Println("Первоначальный путь:")
	grid.PrintPathOnGrid(path)
	PrintPath(path)

	// Обновляем ребро, делая клетку на пути препятствием
	blockedCell1 := grid.Cells[0][2]
	blockedCell2 := grid.Cells[0][3]
	grid.UpdateEdge(blockedCell1, blockedCell2, math.Inf(1))
	fmt.Println("\nСетка после добавления препятствия:")
	grid.PrintGrid()

	path = ExtractPath(grid)
	fmt.Println("Обновленный путь после изменения ребра:")
	PrintPath(path)
	fmt.Println("\nОбновленный путь после изменения ребра:")
	grid.PrintPathOnGrid(path)
}

func TestPathChangesWithFewEdgesUpdates(t *testing.T) {
	start := Point{X: 10, Y: 10}
	goal := Point{X: 14, Y: 14}
	width := 50
	height := 50
	grid := NewGrid(width, height, start, goal)
	fmt.Println("Первоначальная сетка:")
	grid.PrintGrid()
	// Инициализируем соседей для каждой клетки
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := grid.Cells[y][x]
			cell.Neighbors = getNeighbors(x, y, grid.Cells, width, height)
		}
	}

	grid.ComputeShortestPath()

	path := ExtractPath(grid)
	fmt.Println("Первоначальный путь:")
	grid.PrintPathOnGrid(path)
	PrintPath(path)

	// Обновляем ребро, делая клетку на пути препятствием
	//blockedCell1 := grid.Cells[0][2]
	//blockedCell2 := grid.Cells[0][3]
	//grid.UpdateEdge(blockedCell1, blockedCell2, math.Inf(1))

	edge1 := EdgeUpdate{
		U:    grid.Cells[0][2],
		V:    grid.Cells[0][3],
		Cost: math.Inf(1),
	}

	edge2 := EdgeUpdate{
		U:    grid.Cells[0][2],
		V:    grid.Cells[1][2],
		Cost: math.Inf(1),
	}

	edge3 := EdgeUpdate{
		U:    grid.Cells[11][11],
		V:    grid.Cells[12][12],
		Cost: math.Inf(1),
	}

	edge4 := EdgeUpdate{
		U:    grid.Cells[10][2],
		V:    grid.Cells[10][3],
		Cost: math.Inf(1),
	}

	//edges := []EdgeUpdate{
	//	{
	//		U:    nil,
	//		V:    nil,
	//		Cost: 0,
	//	},
	//	{
	//		U:    nil,
	//		V:    nil,
	//		Cost: 0,
	//	},
	//	{
	//		U:    nil,
	//		V:    nil,
	//		Cost: 0,
	//	},
	//	{
	//		U:    nil,
	//		V:    nil,
	//		Cost: 0,
	//	},
	//	{
	//		U:    nil,
	//		V:    nil,
	//		Cost: 0,
	//	},
	//	{
	//		U:    nil,
	//		V:    nil,
	//		Cost: 0,
	//	},
	//}

	grid.UpdateEdges(edge1, edge2, edge3, edge4)
	fmt.Println("\nСетка после добавления препятствия:")
	grid.PrintGrid()

	path = ExtractPath(grid)
	fmt.Println("Обновленный путь после изменения ребра:")
	PrintPath(path)
	fmt.Println("\nОбновленный путь после изменения ребра:")
	grid.PrintPathOnGrid(path)
}
