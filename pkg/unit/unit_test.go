package unit

import (
	"testing"

	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
)

func TestUnitFollowsPathUsingSleepTicks(t *testing.T) {
	u := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	u.SetPath([]geom.Point{
		{X: 24, Y: 8},
		{X: 40, Y: 8},
	})

	u.Tick(1, 1.0/60.0, nil)

	if u.Position != (geom.Point{X: 24, Y: 8}) {
		t.Fatalf("position after first wake = %+v, want %+v", u.Position, geom.Point{X: 24, Y: 8})
	}
	if u.SleepTime() != 20 {
		t.Fatalf("sleepTime = %d, want 20", u.SleepTime())
	}
	if !u.HasPath() {
		t.Fatal("expected remaining path after first logical step")
	}

	for tick := int64(2); tick <= 22; tick++ {
		u.Tick(tick, 1.0/60.0, nil)
	}

	if u.Position != (geom.Point{X: 40, Y: 8}) {
		t.Fatalf("position after second wake = %+v, want %+v", u.Position, geom.Point{X: 40, Y: 8})
	}
	if u.HasPath() {
		t.Fatalf("expected path to be consumed, got %d waypoints", u.PathLen())
	}
}

func TestUnitAppliesSpeedMultiplierToSleepTicks(t *testing.T) {
	u := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	u.SetPath([]geom.Point{{X: 24, Y: 8}})

	u.Tick(1, 1.0/60.0, func(geom.Point) float64 { return 0.5 })

	if u.Position != (geom.Point{X: 24, Y: 8}) {
		t.Fatalf("position = %+v, want %+v", u.Position, geom.Point{X: 24, Y: 8})
	}
	if u.SleepTime() != 40 {
		t.Fatalf("sleepTime = %d, want 40", u.SleepTime())
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

func TestStaticUnitIgnoresPath(t *testing.T) {
	u := NewWall(geom.Point{X: 8, Y: 8})
	u.SetPath([]geom.Point{{X: 24, Y: 8}})
	u.Tick(1, 1.0/60.0, nil)

	if u.Position != (geom.Point{X: 8, Y: 8}) {
		t.Fatalf("position = %+v, want %+v", u.Position, geom.Point{X: 8, Y: 8})
	}
	if u.HasPath() {
		t.Fatal("expected static unit path to be ignored")
	}
}

func TestUnitRenderPositionInterpolatesWhileSleeping(t *testing.T) {
	u := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	u.SetPath([]geom.Point{{X: 24, Y: 8}})

	u.Tick(1, 1.0/60.0, nil)
	if got := u.RenderPosition(); got != (geom.Point{X: 8, Y: 8}) {
		t.Fatalf("render position right after move = %+v, want start point", got)
	}

	for tick := int64(2); tick <= 11; tick++ {
		u.Tick(tick, 1.0/60.0, nil)
	}

	got := u.RenderPosition()
	if !geom.AlmostEqual(got.X, 16) || !geom.AlmostEqual(got.Y, 8) {
		t.Fatalf("render position midway = %+v, want approximately {16 8}", got)
	}
}
