package unit

import (
	"image"
	"math"
	"testing"

	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/world"
)

func TestManagerPanelRectHiddenWhenSelectedUnitOffScreen(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	m := newTestManager(gameWorld,
		NewRunner(geom.Point{X: 16, Y: 16}, false, 0),
	)
	cam := camera.New(camera.Config{})

	m.SelectAtScreen(cam, geom.Point{X: 16, Y: 16}, 64, 64)

	if _, ok := m.PanelRect(cam, 64, 64); !ok {
		t.Fatal("expected panel rect for visible selected unit")
	}

	cam.SetPosition(geom.Point{X: 128, Y: 128})

	if _, ok := m.PanelRect(cam, 64, 64); ok {
		t.Fatal("expected panel rect to be hidden for offscreen selected unit")
	}
}

func TestManagerSelectAtScreenIgnoresOffScreenUnits(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	m := newTestManager(gameWorld,
		NewRunner(geom.Point{X: 160, Y: 160}, false, 0),
	)
	cam := camera.New(camera.Config{})

	m.SelectAtScreen(cam, geom.Point{X: 16, Y: 16}, 64, 64)

	if m.HasSelected() {
		t.Fatal("expected offscreen unit to be ignored by screen selection")
	}
}

func TestManagerSelectAtScreenUsesTileHitInsteadOfSpriteRect(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	m := newTestManager(gameWorld,
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
	)
	cam := camera.New(camera.Config{})

	m.SelectAtScreen(cam, geom.Point{X: 10, Y: 10}, 64, 64)

	if m.HasSelected() {
		t.Fatal("expected click in sprite overhang but outside unit tile to be ignored")
	}

	m.SelectAtScreen(cam, geom.Point{X: 20, Y: 20}, 64, 64)
	if !m.HasSelected() {
		t.Fatal("expected click inside occupied tile to select a unit")
	}
}

func TestManagerSelectAtScreenCanSelectStaticUnit(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	m := newTestManager(gameWorld,
		NewWall(geom.Point{X: 24, Y: 24}),
	)
	cam := camera.New(camera.Config{})

	m.SelectAtScreen(cam, geom.Point{X: 20, Y: 20}, 64, 64)

	if !m.HasSelected() {
		t.Fatal("expected click inside static unit tile to select it")
	}
}

func TestManagerMovesUnitBetweenTileStacksDuringUpdate(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)

	startKey := tileKey{x: 0, y: 0}
	if stack := m.tileStacks[startKey]; stack == nil || len(stack.UnitIDs()) != 1 {
		t.Fatalf("start tile stack = %+v, want one registered unit", stack)
	}

	runner.SetPath([]geom.Point{{X: 24, Y: 8}})
	m.Update(1, 1.0/60.0)

	if stack := m.tileStacks[startKey]; stack != nil && len(stack.UnitIDs()) != 0 {
		t.Fatalf("start tile stack after move = %v, want empty or removed stack", stack.UnitIDs())
	}

	targetKey := tileKey{x: 1, y: 0}
	stack := m.tileStacks[targetKey]
	if stack == nil {
		t.Fatal("expected target tile stack to be created during move")
	}
	if got := stack.UnitIDs(); len(got) != 1 || got[0] != runner.UnitID() {
		t.Fatalf("target tile stack = %v, want runner id %d", got, runner.UnitID())
	}
	if got := m.registeredTiles[runner.UnitID()]; got != targetKey {
		t.Fatalf("registered tile = %+v, want %+v", got, targetKey)
	}
}

func TestManagerVisibleTileUnitsFollowVisibleTileAndTileStackOrder(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	firstInLowerTile := NewRunner(geom.Point{X: 8, Y: 24}, false, 0)
	secondInLowerTile := NewWall(geom.Point{X: 8, Y: 24})
	upperTileUnit := NewWall(geom.Point{X: 24, Y: 8})
	m := newTestManager(gameWorld, firstInLowerTile, secondInLowerTile, upperTileUnit)

	visibleUnits := m.visibleTileUnits(image.Rect(0, 0, 2, 2))
	if len(visibleUnits) != 3 {
		t.Fatalf("visibleTileUnits() len = %d, want 3", len(visibleUnits))
	}

	if visibleUnits[0].UnitID() != upperTileUnit.UnitID() {
		t.Fatalf("visible unit[0] = %d, want upper tile unit %d", visibleUnits[0].UnitID(), upperTileUnit.UnitID())
	}
	if visibleUnits[1].UnitID() != firstInLowerTile.UnitID() {
		t.Fatalf("visible unit[1] = %d, want first lower tile unit %d", visibleUnits[1].UnitID(), firstInLowerTile.UnitID())
	}
	if visibleUnits[2].UnitID() != secondInLowerTile.UnitID() {
		t.Fatalf("visible unit[2] = %d, want second lower tile unit %d", visibleUnits[2].UnitID(), secondInLowerTile.UnitID())
	}
}

func TestManagerVisibleTileUnitsIncludeProjectileFromTileStack(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	m := newTestManager(gameWorld,
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
	)
	m.selectedID = firstOrderedUnitID(t, m.units)

	if err := m.CommandSelectedFire(geom.Point{X: 120, Y: 24}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	projectile := onlyProjectile(t, m)
	startKey := tileKey{x: 1, y: 1}
	if got := m.registeredTiles[projectile.UnitID()]; got != startKey {
		t.Fatalf("registered tile = %+v, want %+v for projectile", got, startKey)
	}

	visibleUnits := m.visibleTileUnits(image.Rect(0, 0, 3, 3))
	if len(visibleUnits) != 2 {
		t.Fatalf("visibleTileUnits() len = %d, want 2 with runner and projectile", len(visibleUnits))
	}
	if visibleUnits[1].UnitID() != projectile.UnitID() {
		t.Fatalf("visible unit[1] = %d, want projectile %d appended in tile stack order", visibleUnits[1].UnitID(), projectile.UnitID())
	}
}

func TestManagerProjectileHitsUnitOccupyingEnteredTile(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	target := NewRunner(geom.Point{X: 37, Y: 28}, false, 0)
	m := newTestManager(gameWorld,
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
		target,
	)
	m.selectedID = firstOrderedUnitID(t, m.units)

	if err := m.CommandSelectedFire(geom.Point{X: 120, Y: 24}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	initialHealth := target.Health
	m.Update(1, 1.0/60.0)

	if target.Health != initialHealth-1 {
		t.Fatalf("target health = %d, want %d after projectile enters occupied tile", target.Health, initialHealth-1)
	}
	projectile := onlyProjectile(t, m)
	if !projectile.exploding {
		t.Fatal("expected projectile to keep living as an explosion after hit")
	}
	if _, ok := m.registeredTiles[projectile.UnitID()]; !ok {
		t.Fatal("expected exploding projectile to stay registered in tile stack until animation ends")
	}

	advanceProjectileExplosion(t, m)
	if projectileCount(m) != 0 {
		t.Fatalf("projectiles = %d, want 0 after explosion animation completes", projectileCount(m))
	}
	if _, ok := m.registeredTiles[projectile.UnitID()]; ok {
		t.Fatal("expected projectile tile registration to be removed after explosion animation completes")
	}
}

func TestManagerProjectileExpiresAfterMaxRange(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 64, Rows: 64, TileSize: 16})
	m := newTestManager(gameWorld,
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
	)
	m.selectedID = firstOrderedUnitID(t, m.units)

	if err := m.CommandSelectedFire(geom.Point{X: 512, Y: 24}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	for range 60 {
		m.Update(1, 1.0/60.0)
	}

	if projectileCount(m) != 0 {
		t.Fatalf("projectiles = %d, want projectile removed after max range", projectileCount(m))
	}
}

func TestManagerProjectileRespawnsUnitAtSpawnPoint(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	target := NewRunner(geom.Point{X: 37, Y: 28}, false, 0)
	target.SpawnPosition = geom.Point{X: 104, Y: 104}
	target.Health = 1

	m := newTestManager(gameWorld,
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
		target,
	)
	m.selectedID = firstOrderedUnitID(t, m.units)

	if err := m.CommandSelectedFire(geom.Point{X: 120, Y: 24}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	m.Update(1, 1.0/60.0)

	if target.Position != target.SpawnPosition {
		t.Fatalf("target position = %+v, want respawn at %+v", target.Position, target.SpawnPosition)
	}
	if target.Health != target.MaxHealth {
		t.Fatalf("target health = %d, want full health %d after respawn", target.Health, target.MaxHealth)
	}
	if projectileCount(m) != 1 {
		t.Fatalf("projectiles = %d, want exploding projectile to remain until animation ends", projectileCount(m))
	}

	advanceProjectileExplosion(t, m)
	if projectileCount(m) != 0 {
		t.Fatalf("projectiles = %d, want projectile removed after explosion animation", projectileCount(m))
	}
}

func TestManagerProjectileCanDamageStaticUnit(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	target := NewWall(geom.Point{X: 37, Y: 28})
	target.Health = 1

	m := newTestManager(gameWorld,
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
		target,
	)
	m.selectedID = firstOrderedUnitID(t, m.units)

	if err := m.CommandSelectedFire(geom.Point{X: 120, Y: 24}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	m.Update(1, 1.0/60.0)

	if target.Position != target.SpawnPosition {
		t.Fatalf("static position = %+v, want respawn at %+v", target.Position, target.SpawnPosition)
	}
	if target.Health != target.MaxHealth {
		t.Fatalf("static health = %d, want full health %d after respawn", target.Health, target.MaxHealth)
	}
}

func TestManagerAssignMoveJobReportsCompletion(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)

	err := m.AssignMoveJob(MoveJob{
		ID:          1,
		ActorID:     7,
		UnitID:      runner.UnitID(),
		TargetTileX: 1,
		TargetTileY: 0,
	})
	if err != nil {
		t.Fatalf("AssignMoveJob() error = %v", err)
	}

	for tick := int64(1); tick <= 200; tick++ {
		m.Update(tick, 1.0/60.0)
		reports := m.DrainJobReports()
		if len(reports) == 0 {
			continue
		}
		if len(reports) != 1 {
			t.Fatalf("DrainJobReports() len = %d, want 1", len(reports))
		}
		if reports[0].Status != JobStatusCompleted {
			t.Fatalf("job status = %v, want %v", reports[0].Status, JobStatusCompleted)
		}
		if reports[0].ActorID != 7 || reports[0].UnitID != runner.UnitID() {
			t.Fatalf("job report = %+v, want actor 7 for unit %d", reports[0], runner.UnitID())
		}
		return
	}

	t.Fatal("expected completed job report within 200 ticks")
}

func TestManagerAssignMoveJobReportsFailureForImmobileTarget(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	wall := NewWall(geom.Point{X: 8, Y: 8})
	m := newTestManager(gameWorld, wall)

	err := m.AssignMoveJob(MoveJob{
		ID:          3,
		ActorID:     11,
		UnitID:      wall.UnitID(),
		TargetTileX: 2,
		TargetTileY: 2,
	})
	if err == nil {
		t.Fatal("AssignMoveJob() error = nil, want immobile-unit error")
	}

	reports := m.DrainJobReports()
	if len(reports) != 1 {
		t.Fatalf("DrainJobReports() len = %d, want 1", len(reports))
	}
	if reports[0].Status != JobStatusFailed {
		t.Fatalf("job status = %v, want %v", reports[0].Status, JobStatusFailed)
	}
	if reports[0].ActorID != 11 || reports[0].UnitID != wall.UnitID() {
		t.Fatalf("job report = %+v, want actor 11 for unit %d", reports[0], wall.UnitID())
	}
}

func TestManagerSkipsStaticUnitUpdateUntilExternalWake(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	target := NewWall(geom.Point{X: 24, Y: 24})
	m := newTestManager(gameWorld, target)

	m.Update(1, 1.0/60.0)
	if target.LastUpdateTick() != 0 {
		t.Fatalf("lastUpdateTick while static unit sleeps = %d, want 0", target.LastUpdateTick())
	}

	target.ApplyDamage(1)
	m.Update(2, 1.0/60.0)
	if target.LastUpdateTick() != 2 {
		t.Fatalf("lastUpdateTick after external wake = %d, want 2", target.LastUpdateTick())
	}

	m.Update(3, 1.0/60.0)
	if target.LastUpdateTick() != 2 {
		t.Fatalf("lastUpdateTick after static unit falls asleep again = %d, want 2", target.LastUpdateTick())
	}
}

func TestManagerCommandSelectedMoveQueuesLatestCommandWhileUnitTravels(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)
	m.selectedID = runner.UnitID()

	if err := m.CommandSelectedMove(2, 0); err != nil {
		t.Fatalf("CommandSelectedMove() initial error = %v", err)
	}

	m.Update(1, 1.0/60.0)
	initialSleep := runner.SleepTime()
	if initialSleep <= 0 {
		t.Fatalf("sleepTime after first move command = %d, want a positive active travel budget", initialSleep)
	}

	if err := m.CommandSelectedMove(1, 1); err != nil {
		t.Fatalf("CommandSelectedMove() queued error = %v", err)
	}
	if err := m.CommandSelectedMove(0, 0); err != nil {
		t.Fatalf("CommandSelectedMove() overwrite queued error = %v", err)
	}
	if runner.SleepTime() != initialSleep {
		t.Fatalf("sleepTime after queued commands = %d, want %d", runner.SleepTime(), initialSleep)
	}

	for tick := int64(2); tick <= int64(initialSleep)+2; tick++ {
		m.Update(tick, 1.0/60.0)
	}

	if runner.Position != (geom.Point{X: 8, Y: 8}) {
		t.Fatalf("position after queued manager command promotion = %+v, want %+v", runner.Position, geom.Point{X: 8, Y: 8})
	}
}

func TestProjectileVisibilityUsesCurrentCameraState(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 64, Rows: 64, TileSize: 16})
	m := newTestManager(gameWorld,
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
	)
	m.selectedID = firstOrderedUnitID(t, m.units)

	if err := m.CommandSelectedFire(geom.Point{X: 120, Y: 24}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	cam := camera.New(camera.Config{})
	var projectile *Projectile
	m.units.Range(func(unit Unit) bool {
		currentProjectile, ok := unit.(*Projectile)
		if !ok {
			return true
		}
		projectile = currentProjectile
		return false
	})
	if projectile == nil {
		t.Fatal("expected projectile to exist")
	}
	if !unitVisibleOnScreen(cam, gameWorld.TileSize(), 64, 64, projectile) {
		t.Fatal("expected projectile to be visible for the default camera")
	}

	cam.SetPosition(geom.Point{X: 400, Y: 400})
	if unitVisibleOnScreen(cam, gameWorld.TileSize(), 64, 64, projectile) {
		t.Fatal("expected projectile to become offscreen after camera move")
	}
}

func projectileCount(m *Manager) int {
	count := 0
	m.units.Range(func(unit Unit) bool {
		if _, ok := unit.(*Projectile); ok {
			count++
		}
		return true
	})
	return count
}

func onlyProjectile(t *testing.T, m *Manager) *Projectile {
	t.Helper()

	var projectile *Projectile
	m.units.Range(func(unit Unit) bool {
		currentProjectile, ok := unit.(*Projectile)
		if !ok {
			return true
		}
		if projectile != nil {
			t.Fatal("expected exactly one projectile")
		}
		projectile = currentProjectile
		return true
	})
	if projectile == nil {
		t.Fatal("expected projectile to exist")
	}

	return projectile
}

func advanceProjectileExplosion(t *testing.T, m *Manager) {
	t.Helper()

	explosionTicks := int(math.Ceil(impactDuration/(1.0/60.0))) + 1
	for tick := int64(0); tick < int64(explosionTicks); tick++ {
		m.Update(100+tick, 1.0/60.0)
	}
}

func firstOrderedUnitID(t *testing.T, units *orderedUnitMap) int64 {
	t.Helper()

	first, ok := units.At(0)
	if !ok || first == nil {
		t.Fatal("expected at least one unit in ordered manager storage")
	}

	return first.UnitID()
}

func newTestManager(gameWorld world.World, units ...Unit) *Manager {
	manager := NewManager(gameWorld)
	for _, current := range units {
		manager.AddUnit(current)
	}
	return manager
}
