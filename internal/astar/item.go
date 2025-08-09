package astar

import (
	"math"

	"github.com/unng-lab/madfarmer/internal/geom"
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
	dx := math.Abs(i.x - goalX)
	dy := math.Abs(i.y - goalY)
	xyMin := math.Min(dx, dy)
	xyMax := math.Max(dx, dy)
	return costDiagonal*xyMin + (xyMax - xyMin)
}

func (i Item) to(targer Item) byte {
	switch x, y := targer.x-i.x, targer.y-i.y; {
	case x == 0 && y == -1:
		return DirUp
	case x == 1 && y == -1:
		return DirUpRight
	case x == 1 && y == 0:
		return DirRight
	case x == 1 && y == 1:
		return DirDownRight
	case x == 0 && y == 1:
		return DirDown
	case x == -1 && y == 1:
		return DirDownLeft
	case x == -1 && y == 0:
		return DirLeft
	case x == -1 && y == -1:
		return DirUpLeft
	default:
		return DirNone
	}
}

const epsilon = 1e-9

func (i Item) Equal(target Item) bool {
	return math.Abs(i.x-target.x) < epsilon
}
