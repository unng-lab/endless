package pathfinding

import "testing"

type testGrid []string

func (g testGrid) InBounds(x, y int) bool {
	return y >= 0 && y < len(g) && x >= 0 && x < len(g[y])
}

func (g testGrid) Cost(x, y int) float64 {
	if !g.InBounds(x, y) {
		return 0
	}
	if g[y][x] == '#' {
		return 0
	}
	return 1
}

func TestFindPath(t *testing.T) {
	grid := testGrid{
		".....",
		".###.",
		"...#.",
		".#...",
		".....",
	}

	path, err := FindPath(grid, Step{X: 0, Y: 0}, Step{X: 4, Y: 4})
	if err != nil {
		t.Fatalf("FindPath returned error: %v", err)
	}
	if len(path) == 0 {
		t.Fatal("FindPath returned empty path")
	}
	if got := path[len(path)-1]; got != (Step{X: 4, Y: 4}) {
		t.Fatalf("FindPath goal = %+v, want %+v", got, Step{X: 4, Y: 4})
	}

	prev := Step{X: 0, Y: 0}
	for _, step := range path {
		dx := step.X - prev.X
		if dx < -1 || dx > 1 {
			t.Fatalf("step jump on X from %+v to %+v", prev, step)
		}
		dy := step.Y - prev.Y
		if dy < -1 || dy > 1 {
			t.Fatalf("step jump on Y from %+v to %+v", prev, step)
		}
		if grid.Cost(step.X, step.Y) <= 0 {
			t.Fatalf("path goes through blocked tile %+v", step)
		}
		prev = step
	}
}

func TestFindPathNoRoute(t *testing.T) {
	grid := testGrid{
		".#.",
		"###",
		".#.",
	}

	_, err := FindPath(grid, Step{X: 0, Y: 0}, Step{X: 2, Y: 2})
	if err != ErrNoPath {
		t.Fatalf("FindPath error = %v, want %v", err, ErrNoPath)
	}
}

func TestFindPathSameStartAndGoal(t *testing.T) {
	grid := testGrid{
		"...",
		"...",
		"...",
	}

	path, err := FindPath(grid, Step{X: 1, Y: 1}, Step{X: 1, Y: 1})
	if err != nil {
		t.Fatalf("FindPath returned error: %v", err)
	}
	if len(path) != 0 {
		t.Fatalf("FindPath len = %d, want 0", len(path))
	}
}
