package pathfinding

import (
	"container/heap"
	"errors"
	"math"
)

var ErrNoPath = errors.New("no path")

type Grid interface {
	InBounds(x, y int) bool
	Cost(x, y int) float64
}

type Step struct {
	X int
	Y int
}

type neighbor struct {
	dx   int
	dy   int
	cost float64
}

var neighbors = [...]neighbor{
	{dx: 0, dy: -1, cost: 1},
	{dx: 1, dy: 0, cost: 1},
	{dx: 0, dy: 1, cost: 1},
	{dx: -1, dy: 0, cost: 1},
	{dx: -1, dy: -1, cost: math.Sqrt2},
	{dx: 1, dy: -1, cost: math.Sqrt2},
	{dx: 1, dy: 1, cost: math.Sqrt2},
	{dx: -1, dy: 1, cost: math.Sqrt2},
}

func FindPath(grid Grid, start, goal Step) ([]Step, error) {
	if !isWalkable(grid, start.X, start.Y) || !isWalkable(grid, goal.X, goal.Y) {
		return nil, ErrNoPath
	}
	if start == goal {
		return nil, nil
	}

	open := priorityQueue{
		&queueItem{
			step:     start,
			priority: heuristic(start, goal),
		},
	}
	heap.Init(&open)

	cameFrom := map[Step]Step{}
	gScore := map[Step]float64{start: 0}
	closed := map[Step]bool{}

	for open.Len() > 0 {
		current := heap.Pop(&open).(*queueItem)
		if closed[current.step] {
			continue
		}
		if current.step == goal {
			return reconstructPath(cameFrom, start, goal), nil
		}
		closed[current.step] = true

		for _, dir := range neighbors {
			next := Step{
				X: current.step.X + dir.dx,
				Y: current.step.Y + dir.dy,
			}
			if !isWalkable(grid, next.X, next.Y) {
				continue
			}
			if dir.dx != 0 && dir.dy != 0 {
				if !isWalkable(grid, current.step.X+dir.dx, current.step.Y) || !isWalkable(grid, current.step.X, current.step.Y+dir.dy) {
					continue
				}
			}

			tileCost := grid.Cost(next.X, next.Y)
			if tileCost <= 0 || math.IsInf(tileCost, 1) {
				continue
			}

			score := gScore[current.step] + dir.cost*tileCost
			if prev, ok := gScore[next]; ok && score >= prev {
				continue
			}

			gScore[next] = score
			cameFrom[next] = current.step
			heap.Push(&open, &queueItem{
				step:     next,
				priority: score + heuristic(next, goal),
			})
		}
	}

	return nil, ErrNoPath
}

func heuristic(from, to Step) float64 {
	dx := math.Abs(float64(from.X - to.X))
	dy := math.Abs(float64(from.Y - to.Y))
	diagonal := math.Min(dx, dy)
	straight := math.Max(dx, dy) - diagonal
	return diagonal*math.Sqrt2 + straight
}

func isWalkable(grid Grid, x, y int) bool {
	return grid.InBounds(x, y) && grid.Cost(x, y) > 0 && !math.IsInf(grid.Cost(x, y), 1)
}

func reconstructPath(cameFrom map[Step]Step, start, goal Step) []Step {
	path := make([]Step, 0, 8)
	for current := goal; current != start; current = cameFrom[current] {
		path = append(path, current)
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path
}

type queueItem struct {
	step     Step
	priority float64
	index    int
}

type priorityQueue []*queueItem

func (pq priorityQueue) Len() int {
	return len(pq)
}

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x any) {
	item := x.(*queueItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	last := len(old) - 1
	item := old[last]
	item.index = -1
	*pq = old[:last]
	return item
}
