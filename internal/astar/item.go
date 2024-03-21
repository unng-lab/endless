package astar

import (
	"github/unng-lab/madfarmer/internal/geom"
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
} // 8 directions

type Item struct {
	x, y     float64
	priority float64
}

func (i Item) heuristic(goalX, goalY float64) float64 {
	return abs(i.x-goalX) + abs(i.y-goalY)
}
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func (i Item) to(targer Item) byte {
	switch x, y := targer.x-i.x, targer.y-i.y; {
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
		panic("unreachable")
	}
}
