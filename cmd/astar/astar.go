package astar

import (
	"container/heap"
	"fmt"
)

type Node struct {
	x, y int
}

type Item struct {
	node     Node
	cost     int
	priority int
}

type PriorityQueue []*Item

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].priority < pq[j].priority }
func (pq PriorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*Item)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

var (
	h, w         int = 6, 6
	start            = Node{0, 0}
	goal             = Node{5, 5}
	costDiagonal     = 2
)

func heuristic(current Node) int {
	return abs(current.x-goal.x) + abs(current.y-goal.y)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func astar() []Node {
	openSet := make(PriorityQueue, 0)
	heap.Init(&openSet)

	cameFrom := make(map[Node]Node)
	gScore := make(map[Node]int)

	heap.Push(&openSet, &Item{start, 0, heuristic(start)})
	gScore[start] = 0

	for len(openSet) > 0 {
		current := heap.Pop(&openSet).(*Item).node

		if current == goal {
			path := make([]Node, 0)
			for current != start {
				path = append(path, current)
				current = cameFrom[current]
			}
			path = append(path, start)
			reversePath(path)
			return path
		}

		for _, offsets := range [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}, {-1, -1}, {-1, 1}, {1, -1}, {1, 1}} {
			neighbor := Node{current.x + offsets[0], current.y + offsets[1]}
			newGScore := gScore[current] + 1
			if offsets[0] != 0 && offsets[1] != 0 {
				newGScore = gScore[current] + costDiagonal
			}

			if oldGScore, ok := gScore[neighbor]; !ok || newGScore < oldGScore {
				gScore[neighbor] = newGScore
				priority := newGScore + heuristic(neighbor)
				heap.Push(&openSet, &Item{neighbor, newGScore, priority})
				cameFrom[neighbor] = current
			}
		}
	}

	return nil
}

func reversePath(path []Node) {
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
}

func main() {
	path := astar()
	for _, node := range path {
		fmt.Printf("(%d, %d) -> ", node.x, node.y)
	}
	fmt.Println("Goal")
}
