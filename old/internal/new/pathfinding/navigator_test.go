package pathfinding

import (
	"testing"

	"github.com/unng-lab/endless/internal/new/tilemap"
)

func TestNavigatorFindsPathThroughPortal(t *testing.T) {
	m := tilemap.New(tilemap.Config{Columns: 128, Rows: 128, TileSize: 1})

	// Create a horizontal wall with a single gap to ensure that the
	// hierarchical search needs to locate a precise portal tile.
	for x := 0; x < m.Columns(); x++ {
		m.SetBlocked(x, 64, true)
	}
	gapX := 80
	m.SetBlocked(gapX, 64, false)

	nav := NewNavigator(m, 16)

	start := Point{X: 10, Y: 10}
	goal := Point{X: 100, Y: 100}

	path, ok := nav.FindPath(start, goal)
	if !ok {
		t.Fatalf("expected to find path")
	}
	if len(path) == 0 {
		t.Fatalf("path should contain at least one step")
	}
	if path[0] != start {
		t.Fatalf("first waypoint must match the start point")
	}
	if path[len(path)-1] != goal {
		t.Fatalf("last waypoint must match the goal point")
	}

	passedGap := false
	for _, p := range path {
		if p.X == gapX && p.Y == 64 {
			passedGap = true
			break
		}
	}
	if !passedGap {
		t.Fatalf("path did not use the only opening in the wall")
	}
}

func TestNavigatorFailsOnBlockedGoal(t *testing.T) {
	m := tilemap.New(tilemap.Config{Columns: 32, Rows: 32, TileSize: 1})
	nav := NewNavigator(m, 8)

	start := Point{X: 1, Y: 1}
	goal := Point{X: 10, Y: 10}
	m.SetBlocked(goal.X, goal.Y, true)

	if path, ok := nav.FindPath(start, goal); ok || path != nil {
		t.Fatalf("expected no path when goal is blocked")
	}
}
