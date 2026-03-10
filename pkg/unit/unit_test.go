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

	u.Tick(1)

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
		if u.SleepTime() > 0 {
			u.StepSleep()
			continue
		}
		u.Tick(tick)
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
	u.SetSpeedMultiplierLookup(func(geom.Point) float64 { return 0.5 })

	u.Tick(1)

	if u.Position != (geom.Point{X: 24, Y: 8}) {
		t.Fatalf("position = %+v, want %+v", u.Position, geom.Point{X: 24, Y: 8})
	}
	if u.SleepTime() != 40 {
		t.Fatalf("sleepTime = %d, want 40", u.SleepTime())
	}
}

func TestUnitVisibleOnScreenReturnsTrueForVisibleUnit(t *testing.T) {
	cam := camera.New(camera.Config{})
	u := NewRunner(geom.Point{X: 16, Y: 16}, false, 0)

	if !unitVisibleOnScreen(cam, 16, 64, 64, u) {
		t.Fatal("expected visible unit to intersect the screen rect")
	}
}

func TestUnitVisibleOnScreenReturnsFalseForOffscreenUnit(t *testing.T) {
	cam := camera.New(camera.Config{})
	u := NewRunner(geom.Point{X: 160, Y: 160}, false, 0)

	if unitVisibleOnScreen(cam, 16, 64, 64, u) {
		t.Fatal("expected offscreen unit to be rejected")
	}
}

func TestRunnerAnimationUsesTickOffsetsAndTickStep(t *testing.T) {
	u := NewRunner(geom.Point{X: 8, Y: 8}, false, 7)

	if got := u.Frame(); got != 1 {
		t.Fatalf("initial frame from tick offset = %d, want 1", got)
	}

	for tick := int64(1); tick <= 5; tick++ {
		u.UpdateVisible(tick)
	}

	if got := u.Frame(); got != 2 {
		t.Fatalf("frame after five visible ticks = %d, want 2", got)
	}
}

func TestStaticUnitIsAlwaysImmobile(t *testing.T) {
	u := NewWall(geom.Point{X: 8, Y: 8})

	if u.IsMobile() {
		t.Fatal("expected wall to remain immobile")
	}
	if !u.BlocksMovement() {
		t.Fatal("expected wall to block movement")
	}
}

func TestStaticUnitSleepsUntilExternalWake(t *testing.T) {
	u := NewWall(geom.Point{X: 8, Y: 8})

	u.Tick(1)
	if u.LastUpdateTick() != 0 {
		t.Fatalf("lastUpdateTick while sleeping = %d, want 0", u.LastUpdateTick())
	}

	u.Wake()
	u.Tick(2)
	if u.LastUpdateTick() != 2 {
		t.Fatalf("lastUpdateTick after wake = %d, want 2", u.LastUpdateTick())
	}

	u.Tick(3)
	if u.LastUpdateTick() != 2 {
		t.Fatalf("lastUpdateTick after returning to sleep = %d, want 2", u.LastUpdateTick())
	}
}

func TestUnitRenderPositionInterpolatesWhileSleeping(t *testing.T) {
	u := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	u.SetPath([]geom.Point{{X: 24, Y: 8}})

	u.Tick(1)
	if got := u.RenderPosition(); got != (geom.Point{X: 8, Y: 8}) {
		t.Fatalf("render position right after move = %+v, want start point", got)
	}

	for tick := int64(2); tick <= 11; tick++ {
		u.StepSleep()
		u.UpdateVisible(tick)
	}

	got := u.RenderPosition()
	if !geom.AlmostEqual(got.X, 16) || !geom.AlmostEqual(got.Y, 8) {
		t.Fatalf("render position midway = %+v, want approximately {16 8}", got)
	}
}

func TestUnitQueueMoveCommandDefersRouteSwitchUntilCurrentTravelCompletes(t *testing.T) {
	u := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	u.SetPath([]geom.Point{
		{X: 24, Y: 8},
		{X: 40, Y: 8},
	})

	u.Tick(1)
	initialSleep := u.SleepTime()

	u.QueueMoveCommand([]geom.Point{{X: 24, Y: 24}})

	if u.SleepTime() != initialSleep {
		t.Fatalf("sleepTime after queued command = %d, want %d while current step is active", u.SleepTime(), initialSleep)
	}
	if destination, ok := u.Destination(); !ok || destination != (geom.Point{X: 40, Y: 8}) {
		t.Fatalf("destination during active step = %+v, %v, want old active route to remain in control", destination, ok)
	}

	for tick := int64(2); tick <= 21; tick++ {
		u.StepSleep()
	}

	if u.Position != (geom.Point{X: 24, Y: 8}) {
		t.Fatalf("position before queued route promotion = %+v, want %+v", u.Position, geom.Point{X: 24, Y: 8})
	}

	u.Tick(22)

	if u.Position != (geom.Point{X: 24, Y: 24}) {
		t.Fatalf("position after queued route promotion = %+v, want %+v", u.Position, geom.Point{X: 24, Y: 24})
	}
	if u.SleepTime() != 20 {
		t.Fatalf("sleepTime after queued route promotion = %d, want 20", u.SleepTime())
	}
}

func TestUnitQueueMoveCommandKeepsOnlyLatestPendingRoute(t *testing.T) {
	u := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	u.SetPath([]geom.Point{
		{X: 24, Y: 8},
		{X: 40, Y: 8},
	})

	u.Tick(1)

	u.QueueMoveCommand([]geom.Point{{X: 24, Y: 24}})
	u.QueueMoveCommand([]geom.Point{{X: 8, Y: 8}})

	for tick := int64(2); tick <= 21; tick++ {
		u.StepSleep()
	}

	u.Tick(22)

	if u.Position != (geom.Point{X: 8, Y: 8}) {
		t.Fatalf("position after overwriting pending route = %+v, want %+v", u.Position, geom.Point{X: 8, Y: 8})
	}
}
