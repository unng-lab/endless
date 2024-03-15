package astar

type Item struct {
	x, y     int
	priority int
}

func (i Item) heuristic(goalX, goalY int) int {
	return abs(i.x-goalX) + abs(i.y-goalY)
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
