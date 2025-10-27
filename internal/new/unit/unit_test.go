package unit

import (
	"testing"

	"github.com/unng-lab/endless/internal/new/camera"
	"github.com/unng-lab/endless/internal/new/pathfinding"
	"github.com/unng-lab/endless/internal/new/tilemap"
)

func TestUnitsFollowAssignedPath(t *testing.T) {
	m := tilemap.New(tilemap.Config{Columns: 64, Rows: 64, TileSize: 2})
	nav := pathfinding.NewNavigator(m, 8)
	mgr := NewManager(m, nav)

	worker := &UnitType{
		Name:  "worker",
		Speed: 10,
		Animations: map[State]Animation{
			StateIdle:   {Frames: []int{1}, FrameDuration: 0.5},
			StateMoving: {Frames: []int{2, 3}, FrameDuration: 0.25},
		},
	}

	start := camera.Point{X: 1, Y: 1}
	u := NewUnit(worker, start)
	mgr.Add(u)

	target := pathfinding.Point{X: 40, Y: 40}
	if err := mgr.CommandMove(u, target); err != nil {
		t.Fatalf("unexpected error assigning path: %v", err)
	}

	// Simulate updates until the unit reaches the goal.
	for i := 0; i < 5000 && u.State() != StateIdle; i++ {
		mgr.Update(0.05)
	}

	if u.State() != StateIdle {
		t.Fatalf("unit should be idle after completing the path")
	}

	tile := mgr.worldToTile(u.Position)
	if tile != target {
		t.Fatalf("expected unit to stop at %v, got %v", target, tile)
	}

	frame := u.FrameIndex()
	if frame != worker.Animations[StateIdle].Frames[0] {
		t.Fatalf("expected idle animation frame after path completion")
	}
}

func TestMultipleUnitTypesHaveIndependentAnimations(t *testing.T) {
	m := tilemap.New(tilemap.Config{Columns: 16, Rows: 16, TileSize: 4})
	nav := pathfinding.NewNavigator(m, 4)
	mgr := NewManager(m, nav)

	rogue := &UnitType{
		Name:  "rogue",
		Speed: 12,
		Animations: map[State]Animation{
			StateIdle:   {Frames: []int{5}, FrameDuration: 0.3},
			StateMoving: {Frames: []int{6, 7, 8}, FrameDuration: 0.1},
		},
	}
	knight := &UnitType{
		Name:  "knight",
		Speed: 8,
		Animations: map[State]Animation{
			StateIdle:   {Frames: []int{9}, FrameDuration: 0.3},
			StateMoving: {Frames: []int{10, 11}, FrameDuration: 0.2},
		},
	}

	r := NewUnit(rogue, camera.Point{X: 8, Y: 8})
	k := NewUnit(knight, camera.Point{X: 16, Y: 8})
	mgr.Add(r)
	mgr.Add(k)

	if err := mgr.CommandMove(r, pathfinding.Point{X: 12, Y: 12}); err != nil {
		t.Fatalf("command move rogue: %v", err)
	}
	if err := mgr.CommandMove(k, pathfinding.Point{X: 14, Y: 14}); err != nil {
		t.Fatalf("command move knight: %v", err)
	}

	mgr.Update(0.2)

	rFrame := r.FrameIndex()
	kFrame := k.FrameIndex()

	if rFrame == kFrame {
		t.Fatalf("expected different animation frames for different unit types")
	}
}
