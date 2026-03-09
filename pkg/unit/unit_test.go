package unit

import (
	"testing"

	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
)

func TestUnitFollowsPath(t *testing.T) {
	u := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	u.SetPath([]geom.Point{
		{X: 24, Y: 8},
		{X: 40, Y: 8},
	})

	u.Update(1, nil)

	if u.Position != (geom.Point{X: 40, Y: 8}) {
		t.Fatalf("position = %+v, want %+v", u.Position, geom.Point{X: 40, Y: 8})
	}
	if u.HasPath() {
		t.Fatalf("expected path to be consumed, got %d waypoints", u.PathLen())
	}
}

func TestUnitAppliesSpeedMultiplier(t *testing.T) {
	u := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	u.SetPath([]geom.Point{{X: 24, Y: 8}})

	u.Update(0.5, func(geom.Point) float64 { return 0.5 })

	if u.Position != (geom.Point{X: 20, Y: 8}) {
		t.Fatalf("position = %+v, want %+v", u.Position, geom.Point{X: 20, Y: 8})
	}
	if !u.HasPath() {
		t.Fatal("expected remaining path after slow movement")
	}
}

func TestStaticUnitIgnoresPath(t *testing.T) {
	u := NewWall(geom.Point{X: 8, Y: 8})
	u.SetPath([]geom.Point{{X: 24, Y: 8}})
	u.Update(1, nil)

	if u.Position != (geom.Point{X: 8, Y: 8}) {
		t.Fatalf("position = %+v, want %+v", u.Position, geom.Point{X: 8, Y: 8})
	}
	if u.HasPath() {
		t.Fatal("expected static unit path to be ignored")
	}
}

func TestUpdateOnScreenMarksVisibleUnit(t *testing.T) {
	cam := camera.New(camera.Config{})
	u := NewRunner(geom.Point{X: 16, Y: 16}, false, 0)

	UpdateOnScreen(cam, 16, 64, 64, &u)

	if !u.OnScreen {
		t.Fatal("expected unit to be marked as visible")
	}
}

func TestUpdateOnScreenMarksOffscreenUnit(t *testing.T) {
	cam := camera.New(camera.Config{})
	u := NewRunner(geom.Point{X: 160, Y: 160}, false, 0)

	UpdateOnScreen(cam, 16, 64, 64, &u)

	if u.OnScreen {
		t.Fatal("expected unit to be marked as offscreen")
	}
}
