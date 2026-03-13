package unit

import (
	"bytes"
	"image"
	"log"
	"strings"
	"testing"

	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/world"
)

type tickOrderExpectation struct {
	sleepTime      int
	pathLen        int
	activeOrderID  int64
	activeKind     OrderKind
	activeTarget   geom.Point
	queuedOrderID  int64
	queuedKind     OrderKind
	queuedTarget   geom.Point
	queuedPathLen  int
	reachedTileX   int
	reachedTileY   int
	lastUpdateTick int64
}

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

// TestManagerSelectUnitByIDDoesNotInterruptSleepingMove reproduces the visual RL duel pattern
// where scenario code keeps the shooter selected every tick while a move order is already in
// flight. Selection is UI-only state and must not reset the sleep budget of the active segment.
func TestManagerSelectUnitByIDDoesNotInterruptSleepingMove(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)

	if err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 40, Y: 8}); err != nil {
		t.Fatalf("IssueMoveOrder() error = %v", err)
	}

	m.Update(1)
	initialSleep := runner.SleepTime()
	if initialSleep <= 0 {
		t.Fatalf("sleepTime after first update = %d, want positive travel budget", initialSleep)
	}
	if got := runner.Position; got != (geom.Point{X: 24, Y: 8}) {
		t.Fatalf("position after first update = %+v, want %+v", got, geom.Point{X: 24, Y: 8})
	}

	for tick := int64(2); tick < int64(initialSleep)+1; tick++ {
		if !m.SelectUnitByID(runner.UnitID()) {
			t.Fatalf("SelectUnitByID(%d) = false, want true", runner.UnitID())
		}

		m.Update(tick)

		if got := runner.Position; got != (geom.Point{X: 24, Y: 8}) {
			t.Fatalf("position during sleeping travel at tick %d = %+v, want %+v", tick, got, geom.Point{X: 24, Y: 8})
		}
		if got := runner.LastUpdateTick(); got != 1 {
			t.Fatalf("lastUpdateTick during sleeping travel at tick %d = %d, want 1", tick, got)
		}
	}

	wakeTick := int64(initialSleep) + 1
	if !m.SelectUnitByID(runner.UnitID()) {
		t.Fatalf("SelectUnitByID(%d) at wake tick = false, want true", runner.UnitID())
	}

	m.Update(wakeTick)

	if got := runner.Position; got != (geom.Point{X: 40, Y: 8}) {
		t.Fatalf("position after wake tick = %+v, want %+v", got, geom.Point{X: 40, Y: 8})
	}
	if got := runner.LastUpdateTick(); got != wakeTick {
		t.Fatalf("lastUpdateTick after wake tick = %d, want %d", got, wakeTick)
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
	m.Update(1)

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

func TestManagerRunsNextTickImmediatelyAfterUnitReachesNextTile(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)

	runner.SetPath([]geom.Point{
		{X: 24, Y: 8},
		{X: 40, Y: 8},
	})

	m.Update(1)
	if got := runner.Position; got != (geom.Point{X: 24, Y: 8}) {
		t.Fatalf("position after first update = %+v, want %+v", got, geom.Point{X: 24, Y: 8})
	}
	initialSleep := runner.SleepTime()
	if initialSleep <= 0 {
		t.Fatalf("sleepTime after first update = %d, want positive travel budget", initialSleep)
	}
	wakeTick := int64(initialSleep) + 1

	for tick := int64(2); tick < wakeTick; tick++ {
		m.Update(tick)
	}

	if got := runner.Position; got != (geom.Point{X: 24, Y: 8}) {
		t.Fatalf("position before wake tick = %+v, want %+v", got, geom.Point{X: 24, Y: 8})
	}
	if got := runner.LastUpdateTick(); got != 1 {
		t.Fatalf("lastUpdateTick before wake tick = %d, want 1", got)
	}

	m.Update(wakeTick)

	if got := runner.Position; got != (geom.Point{X: 40, Y: 8}) {
		t.Fatalf("position on wake tick = %+v, want %+v", got, geom.Point{X: 40, Y: 8})
	}
	if got := runner.LastUpdateTick(); got != wakeTick {
		t.Fatalf("lastUpdateTick on wake tick = %d, want %d", got, wakeTick)
	}
}

func TestManagerVisibleTileUnitsFollowVisibleTileAndTileStackOrder(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	firstInLowerTile := NewRunner(geom.Point{X: 8, Y: 24}, false, 0)
	secondInLowerTile := NewWall(geom.Point{X: 8, Y: 24})
	upperTileUnit := NewWall(geom.Point{X: 24, Y: 8})
	m := newTestManager(gameWorld, firstInLowerTile, secondInLowerTile, upperTileUnit)

	visibleUnits := m.visibleTileUnits(image.Rect(0, 0, 2, 2), true)
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

	advanceFireOrderUntilProjectileSpawned(t, m, 1)
	projectile := onlyProjectile(t, m)
	startKey := tileKey{x: 1, y: 1}
	if got := m.registeredTiles[projectile.UnitID()]; got != startKey {
		t.Fatalf("registered tile = %+v, want %+v for projectile", got, startKey)
	}

	visibleUnits := m.visibleTileUnits(image.Rect(0, 0, 3, 3), true)
	if len(visibleUnits) != 2 {
		t.Fatalf("visibleTileUnits() len = %d, want 2 with runner and projectile", len(visibleUnits))
	}
	if visibleUnits[1].UnitID() != projectile.UnitID() {
		t.Fatalf("visible unit[1] = %d, want projectile %d appended in tile stack order", visibleUnits[1].UnitID(), projectile.UnitID())
	}
}

func TestManagerVisibleTileUnitsAdvanceOnlyVisibleUnitRenderState(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 64, Rows: 64, TileSize: 16})
	visibleRunner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	hiddenRunner := NewRunner(geom.Point{X: 328, Y: 8}, false, 0)
	m := newTestManager(gameWorld, visibleRunner, hiddenRunner)

	visibleRunner.SetPath([]geom.Point{{X: 24, Y: 8}})
	hiddenRunner.SetPath([]geom.Point{{X: 344, Y: 8}})
	m.Update(1)
	m.Update(2)

	visibleUnits := m.visibleTileUnits(image.Rect(0, 0, 2, 2), true)
	if len(visibleUnits) != 1 || visibleUnits[0].UnitID() != visibleRunner.UnitID() {
		t.Fatalf("visibleTileUnits() = %v, want only visible runner", len(visibleUnits))
	}

	visibleRender := visibleRunner.RenderPosition()
	if !(visibleRender.X > 8 && visibleRender.X < 24) {
		t.Fatalf("visible runner render position = %+v, want interpolation between tiles", visibleRender)
	}

	hiddenRender := hiddenRunner.RenderPosition()
	if hiddenRender != (geom.Point{X: 328, Y: 8}) {
		t.Fatalf("hidden runner render position = %+v, want untouched hidden start point", hiddenRender)
	}
}

func TestManagerVisibleTileUnitsCanSkipAdditionalVisibleRefresh(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 64, Rows: 64, TileSize: 16})
	visibleRunner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, visibleRunner)

	visibleRunner.SetPath([]geom.Point{{X: 24, Y: 8}})
	m.Update(1)
	m.Update(2)

	visibleUnits := m.visibleTileUnits(image.Rect(0, 0, 2, 2), false)
	if len(visibleUnits) != 1 || visibleUnits[0].UnitID() != visibleRunner.UnitID() {
		t.Fatalf("visibleTileUnits() = %v, want only visible runner", len(visibleUnits))
	}

	if got := visibleRunner.RenderPosition(); got != (geom.Point{X: 8, Y: 8}) {
		t.Fatalf("render position without additional visible refresh = %+v, want unchanged start point", got)
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

	spawnTick := advanceFireOrderUntilProjectileSpawned(t, m, 1)
	initialHealth := target.Health
	m.Update(spawnTick + 1)

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

	spawnTick := advanceFireOrderUntilProjectileSpawned(t, m, 1)
	for tick := spawnTick + 1; tick <= spawnTick+60; tick++ {
		m.Update(tick)
	}

	if projectileCount(m) != 0 {
		t.Fatalf("projectiles = %d, want projectile removed after max range", projectileCount(m))
	}
}

func TestManagerProjectileRemovesKilledUnitFromManager(t *testing.T) {
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

	spawnTick := advanceFireOrderUntilProjectileSpawned(t, m, 1)
	m.Update(spawnTick + 1)

	if projectileCount(m) != 1 {
		t.Fatalf("projectiles = %d, want exploding projectile to remain until animation ends", projectileCount(m))
	}
	if _, ok := m.unitByID(target.UnitID()); ok {
		t.Fatalf("unitByID(%d) = true, want killed target removed", target.UnitID())
	}
	if _, ok := m.registeredTiles[target.UnitID()]; ok {
		t.Fatalf("registeredTiles contains %d, want removed target to be unregistered", target.UnitID())
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

	spawnTick := advanceFireOrderUntilProjectileSpawned(t, m, 1)
	m.Update(spawnTick + 1)

	if _, ok := m.unitByID(target.UnitID()); ok {
		t.Fatalf("unitByID(%d) = true, want killed static unit removed", target.UnitID())
	}
	if _, ok := m.registeredTiles[target.UnitID()]; ok {
		t.Fatalf("registeredTiles contains %d, want removed static unit to be unregistered", target.UnitID())
	}
}

func TestManagerIssueMoveOrderReportsQueuedStartedAndCompleted(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)

	err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 24, Y: 8})
	if err != nil {
		t.Fatalf("IssueMoveOrder() error = %v", err)
	}

	collected := make([]OrderReport, 0)
	for tick := int64(1); tick <= 200; tick++ {
		m.Update(tick)
		reports := m.DrainUnitOrderReports(runner.UnitID())
		if len(reports) == 0 {
			continue
		}
		collected = append(collected, reports...)
		if !containsOrderStatus(collected, OrderCompleted) {
			continue
		}

		assertOrderStatusesPresent(t, collected, OrderQueued, OrderStarted, OrderCompleted)
		return
	}

	t.Fatal("expected completed move order report within 200 ticks")
}

func TestManagerIssueMoveOrderReportsFailureForImmobileTarget(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	wall := NewWall(geom.Point{X: 8, Y: 8})
	m := newTestManager(gameWorld, wall)

	err := m.IssueMoveOrder(wall.UnitID(), geom.Point{X: 40, Y: 40})
	if err == nil {
		t.Fatal("IssueMoveOrder() error = nil, want immobile-unit error")
	}

	reports := m.DrainUnitOrderReports(wall.UnitID())
	if len(reports) != 1 {
		t.Fatalf("DrainUnitOrderReports() len = %d, want 1", len(reports))
	}
	if reports[0].Status != OrderFailed {
		t.Fatalf("order status = %v, want %v", reports[0].Status, OrderFailed)
	}
	if reports[0].Kind != OrderKindMove || reports[0].UnitID != wall.UnitID() {
		t.Fatalf("order report = %+v, want move failure for unit %d", reports[0], wall.UnitID())
	}
}

func TestManagerIssueMoveOrderFailsWhenDestinationTileHasBlockingUnit(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	blocker := NewWall(geom.Point{X: 24, Y: 8})
	m := newTestManager(gameWorld, runner, blocker)

	err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 24, Y: 8})
	if err == nil {
		t.Fatal("IssueMoveOrder() error = nil, want blocker-related path failure")
	}

	reports := m.DrainUnitOrderReports(runner.UnitID())
	if len(reports) != 1 {
		t.Fatalf("DrainUnitOrderReports() len = %d, want 1", len(reports))
	}
	if reports[0].Status != OrderFailed {
		t.Fatalf("order status = %v, want %v", reports[0].Status, OrderFailed)
	}
}

func TestManagerCollectsCanceledOrderBeforeRemovingDeadUnit(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)

	err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 24, Y: 8})
	if err != nil {
		t.Fatalf("IssueMoveOrder() error = %v", err)
	}

	if !runner.ApplyDamage(runner.MaxHealth) {
		t.Fatal("ApplyDamage() = false, want lethal damage")
	}

	m.Update(1)

	reports := m.DrainUnitOrderReports(runner.UnitID())
	assertOrderStatusesPresent(t, reports, OrderQueued, OrderCanceled)
	if reports[len(reports)-1].UnitID != runner.UnitID() {
		t.Fatalf("last order report = %+v, want unit %d", reports[len(reports)-1], runner.UnitID())
	}
	if _, ok := m.unitByID(runner.UnitID()); ok {
		t.Fatalf("unitByID(%d) = true, want dead runner removed", runner.UnitID())
	}
}

func TestManagerDrainUnitOrderReportsKeepsStatusesScopedToRequestedUnit(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	firstRunner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	secondRunner := NewRunner(geom.Point{X: 40, Y: 8}, false, 0)
	m := newTestManager(gameWorld, firstRunner, secondRunner)

	err := m.IssueMoveOrder(firstRunner.UnitID(), geom.Point{X: 24, Y: 8})
	if err != nil {
		t.Fatalf("IssueMoveOrder() error = %v", err)
	}

	collected := make([]OrderReport, 0)
	for tick := int64(1); tick <= 200; tick++ {
		m.Update(tick)
		if len(m.DrainUnitOrderReports(secondRunner.UnitID())) != 0 {
			t.Fatal("expected unrelated unit to have no order reports")
		}

		reports := m.DrainUnitOrderReports(firstRunner.UnitID())
		if len(reports) == 0 {
			continue
		}
		collected = append(collected, reports...)
		if !containsOrderStatus(collected, OrderCompleted) {
			continue
		}

		assertOrderStatusesPresent(t, collected, OrderQueued, OrderStarted, OrderCompleted)
		return
	}

	t.Fatal("expected completed order report for requested unit within 200 ticks")
}

func TestManagerIssueFireOrderReportsQueuedStartedAndCompleted(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 64, Rows: 64, TileSize: 16})
	runner := NewRunner(geom.Point{X: 24, Y: 24}, false, 0)
	m := newTestManager(gameWorld, runner)

	if err := m.IssueFireOrder(runner.UnitID(), geom.Point{X: 1, Y: 0}); err != nil {
		t.Fatalf("IssueFireOrder() error = %v", err)
	}

	collected := make([]OrderReport, 0)
	for tick := int64(1); tick <= 32; tick++ {
		m.Update(tick)
		collected = append(collected, m.DrainUnitOrderReports(runner.UnitID())...)
		if !containsOrderStatus(collected, OrderCompleted) {
			continue
		}

		assertOrderStatusesPresent(t, collected, OrderQueued, OrderStarted, OrderCompleted)
		if projectileCount(m) != 1 {
			t.Fatalf("projectiles = %d, want one released projectile after completed fire order", projectileCount(m))
		}
		return
	}

	t.Fatal("expected completed fire order report within 32 ticks")
}

func TestManagerIssueFireOrderWaitsForCooldownBeforeStartingNextShot(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 64, Rows: 64, TileSize: 16})
	runner := NewRunner(geom.Point{X: 24, Y: 24}, false, 0)
	m := newTestManager(gameWorld, runner)

	if err := m.IssueFireOrder(runner.UnitID(), geom.Point{X: 1, Y: 0}); err != nil {
		t.Fatalf("IssueFireOrder() first error = %v", err)
	}

	firstCompleteTick := int64(0)
	for tick := int64(1); tick <= 32; tick++ {
		m.Update(tick)
		reports := m.DrainUnitOrderReports(runner.UnitID())
		if !containsOrderStatus(reports, OrderCompleted) {
			continue
		}

		firstCompleteTick = tick
		break
	}
	if firstCompleteTick == 0 {
		t.Fatal("expected first fire order to complete")
	}

	if err := m.IssueFireOrder(runner.UnitID(), geom.Point{X: 1, Y: 0}); err != nil {
		t.Fatalf("IssueFireOrder() second error = %v", err)
	}

	queuedReports := m.DrainUnitOrderReports(runner.UnitID())
	if len(queuedReports) != 1 || queuedReports[0].Status != OrderQueued {
		t.Fatalf("queued fire reports = %+v, want one queued second order", queuedReports)
	}
	secondOrderID := queuedReports[0].OrderID

	for tick := firstCompleteTick + 1; tick < firstCompleteTick+fireOrderCooldownTicks; tick++ {
		m.Update(tick)
		reports := reportsForOrderID(m.DrainUnitOrderReports(runner.UnitID()), secondOrderID)
		if containsOrderStatus(reports, OrderStarted) {
			t.Fatalf("second fire order started at tick %d before cooldown expired", tick)
		}
	}

	startTick := firstCompleteTick + fireOrderCooldownTicks
	m.Update(startTick)
	reports := reportsForOrderID(m.DrainUnitOrderReports(runner.UnitID()), secondOrderID)
	if !containsOrderStatus(reports, OrderStarted) {
		t.Fatalf("second fire order reports at tick %d = %+v, want started after cooldown", startTick, reports)
	}
}

func TestManagerExternalAPIDebugLoggingStaysDisabledByDefault(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)

	var buffer bytes.Buffer
	previousWriter := log.Writer()
	log.SetOutput(&buffer)
	defer log.SetOutput(previousWriter)
	buffer.Reset()

	if err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 24, Y: 8}); err != nil {
		t.Fatalf("IssueMoveOrder() error = %v", err)
	}
	if err := m.IssueFireOrder(runner.UnitID(), geom.Point{X: 1, Y: 0}); err != nil {
		t.Fatalf("IssueFireOrder() error = %v", err)
	}

	if buffer.Len() != 0 {
		t.Fatalf("debug log output = %q, want no output while debug logging is disabled", buffer.String())
	}
}

func TestManagerExternalAPIDebugLoggingWritesMoveAndFireCalls(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)
	m.SetExternalAPIDebugLogging(true)

	var buffer bytes.Buffer
	previousWriter := log.Writer()
	log.SetOutput(&buffer)
	defer log.SetOutput(previousWriter)
	buffer.Reset()

	if err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 24, Y: 8}); err != nil {
		t.Fatalf("IssueMoveOrder() error = %v", err)
	}
	if err := m.IssueFireOrder(runner.UnitID(), geom.Point{X: 1, Y: 0}); err != nil {
		t.Fatalf("IssueFireOrder() error = %v", err)
	}

	output := buffer.String()
	if !strings.Contains(output, "[debug][external-api] IssueMoveOrder move") {
		t.Fatalf("move debug log missing from %q", output)
	}
	if !strings.Contains(output, "accepted=true") {
		t.Fatalf("accepted flag missing from %q", output)
	}
	if !strings.Contains(output, "path_waypoints=") {
		t.Fatalf("move path details missing from %q", output)
	}
	if !strings.Contains(output, "[debug][external-api] fire unit=") {
		t.Fatalf("fire debug log missing from %q", output)
	}
	if !strings.Contains(output, "normalized=(1.000, 0.000)") {
		t.Fatalf("normalized fire direction missing from %q", output)
	}
}

func TestManagerExternalAPIDebugLoggingIncludesUnitRuntimeQueueAndMoveTrace(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)
	m.SetExternalAPIDebugLogging(true)

	var buffer bytes.Buffer
	previousWriter := log.Writer()
	log.SetOutput(&buffer)
	defer log.SetOutput(previousWriter)
	buffer.Reset()

	if err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 40, Y: 8}); err != nil {
		t.Fatalf("IssueMoveOrder() initial error = %v", err)
	}
	m.Update(1)
	if err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 56, Y: 8}); err != nil {
		t.Fatalf("IssueMoveOrder() queued error = %v", err)
	}
	for tick := int64(2); tick <= 64; tick++ {
		m.Update(tick)
	}

	output := buffer.String()
	if !strings.Contains(output, "[debug][unit-runtime] queue") {
		t.Fatalf("unit queue debug log missing from %q", output)
	}
	if !strings.Contains(output, "[debug][unit-runtime] start") {
		t.Fatalf("unit start debug log missing from %q", output)
	}
	if !strings.Contains(output, "[debug][unit-runtime] move-step") {
		t.Fatalf("unit move-step debug log missing from %q", output)
	}
	if !strings.Contains(output, "[debug][unit-runtime] handoff") {
		t.Fatalf("unit handoff debug log missing from %q", output)
	}
}

func TestManagerExternalAPIDebugLoggingKeepsMoveOriginAtLastReachedTileDuringTravel(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)
	m.SetExternalAPIDebugLogging(true)

	var buffer bytes.Buffer
	previousWriter := log.Writer()
	log.SetOutput(&buffer)
	defer log.SetOutput(previousWriter)
	buffer.Reset()

	if err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 40, Y: 8}); err != nil {
		t.Fatalf("IssueMoveOrder() initial error = %v", err)
	}

	m.Update(1)
	buffer.Reset()

	if err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 56, Y: 8}); err != nil {
		t.Fatalf("IssueMoveOrder() queued error = %v", err)
	}

	output := buffer.String()
	if !strings.Contains(output, "from_tile=(0, 0)") {
		t.Fatalf("queued move log = %q, want last reached tile origin", output)
	}
	if !strings.Contains(output, "accepted=true") {
		t.Fatalf("queued move log = %q, want accepted=true in debug output", output)
	}
}

func TestManagerFlushesTileLeaveForLastDeletedUnit(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	runner := NewRunner(geom.Point{X: 8, Y: 8}, false, 0)
	m := newTestManager(gameWorld, runner)

	if !runner.ApplyDamage(runner.MaxHealth) {
		t.Fatal("ApplyDamage() = false, want lethal damage")
	}

	m.Update(1)

	if _, ok := m.registeredTiles[runner.UnitID()]; ok {
		t.Fatalf("registeredTiles contains %d, want deleted last unit removed from tile registry", runner.UnitID())
	}
	if _, ok := m.unitByID(runner.UnitID()); ok {
		t.Fatalf("unitByID(%d) = true, want deleted last unit hidden from lookup", runner.UnitID())
	}
}

func TestManagerSkipsStaticUnitUpdateUntilExternalWake(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	target := NewWall(geom.Point{X: 24, Y: 24})
	m := newTestManager(gameWorld, target)

	m.Update(1)
	if target.LastUpdateTick() != 0 {
		t.Fatalf("lastUpdateTick while static unit sleeps = %d, want 0", target.LastUpdateTick())
	}

	target.ApplyDamage(1)
	m.Update(2)
	if target.LastUpdateTick() != 2 {
		t.Fatalf("lastUpdateTick after external wake = %d, want 2", target.LastUpdateTick())
	}

	m.Update(3)
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

	m.Update(1)
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
		m.Update(tick)
	}

	if runner.Position != (geom.Point{X: 8, Y: 8}) {
		t.Fatalf("position after queued manager command promotion = %+v, want %+v", runner.Position, geom.Point{X: 8, Y: 8})
	}
}

// TestManagerIssueMoveOrderQueuesRapidExternalReroutesUntilNextTileCenter reproduces the same
// external-api shape as the reported gameplay issue: one long move is already in flight and a
// later IssueMoveOrder arrives before the unit reaches the next tile center. The new move order
// must be accepted immediately, but it may only remain queued until the current movement segment
// has actually reached the next center.
func TestManagerIssueMoveOrderQueuesRapidExternalReroutesUntilNextTileCenter(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 10000, Rows: 10000, TileSize: 16})
	runner := NewRunner(geom.Point{
		X: (4990.0 + 0.5) * gameWorld.TileSize(),
		Y: (5000.0 + 0.5) * gameWorld.TileSize(),
	}, false, 0)
	m := newTestManager(gameWorld, runner)

	if err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 79994.0, Y: 80032.3}); err != nil {
		t.Fatalf("IssueMoveOrder() initial error = %v", err)
	}

	for tick := int64(1); tick <= 10; tick++ {
		m.Update(tick)
	}

	firstSnapshot, ok := m.UnitSnapshot(runner.UnitID())
	if !ok {
		t.Fatal("UnitSnapshot() after tick 10 = false, want live runner snapshot")
	}
	if firstSnapshot.SleepTime <= 0 {
		t.Fatalf("snapshot after tick 10 sleepTime = %d, want active in-flight segment", firstSnapshot.SleepTime)
	}
	if firstSnapshot.TileX != 4990 || firstSnapshot.TileY != 5000 {
		t.Fatalf(
			"snapshot tile after tick 10 = (%d, %d), want last reached tile (4990, 5000) while segment sleep is still active",
			firstSnapshot.TileX,
			firstSnapshot.TileY,
		)
	}

	if err := m.IssueMoveOrder(runner.UnitID(), geom.Point{X: 79992.0, Y: 79992.0}); err != nil {
		t.Fatalf("IssueMoveOrder() reroute at tick 10 error = %v", err)
	}

	reports := m.DrainUnitOrderReports(runner.UnitID())
	moveQueued := false
	for _, report := range reports {
		if report.Kind == OrderKindMove && report.Status == OrderQueued {
			moveQueued = true
			break
		}
	}
	if !moveQueued {
		t.Fatalf("order reports %+v do not contain a queued move reroute", reports)
	}

	for tick := int64(11); tick <= 14; tick++ {
		m.Update(tick)
	}

	secondSnapshot, ok := m.UnitSnapshot(runner.UnitID())
	if !ok {
		t.Fatal("UnitSnapshot() after tick 14 = false, want live runner snapshot")
	}
	if !secondSnapshot.HasQueuedMoveOrder {
		t.Fatal("snapshot after tick 14 should keep the newer reroute queued while the current segment is unfinished")
	}
	if secondSnapshot.TileX != 4990 || secondSnapshot.TileY != 5000 {
		t.Fatalf(
			"snapshot tile after tick 14 = (%d, %d), want last reached tile (4990, 5000) until the first segment reaches the next cell center",
			secondSnapshot.TileX,
			secondSnapshot.TileY,
		)
	}

	rerouteReports := make([]OrderReport, 0)
	rerouteStartTick := int64(14 + secondSnapshot.SleepTime)
	for tick := int64(15); tick < rerouteStartTick; tick++ {
		m.Update(tick)
		rerouteReports = append(rerouteReports, reportsForTargetPoint(m.DrainUnitOrderReports(runner.UnitID()), geom.Point{X: 79992.0, Y: 79992.0})...)
		if containsOrderStatus(rerouteReports, OrderStarted) {
			t.Fatalf("reroute reports = %+v, want queued move to stay pending before tick %d", rerouteReports, rerouteStartTick)
		}
	}

	m.Update(rerouteStartTick)
	rerouteReports = append(rerouteReports, reportsForTargetPoint(m.DrainUnitOrderReports(runner.UnitID()), geom.Point{X: 79992.0, Y: 79992.0})...)
	if !containsOrderStatus(rerouteReports, OrderStarted) {
		t.Fatalf(
			"reroute reports at tick %d = %+v, want queued move to start exactly when the current segment reaches the next tile center",
			rerouteStartTick,
			rerouteReports,
		)
	}
}

// TestManagerIssueSequenceKeepsInternalOrderStateStableAcrossLongRerouteChain replays the longer
// external command sequence from the reported gameplay log and inspects the unit internals on
// every tick. The important contract is that repeated Issue... calls may replace only the queued
// order while the current movement segment is still sleeping; the active path and active order
// must stay unchanged until the unit reaches the next tile center.
func TestManagerIssueSequenceKeepsInternalOrderStateStableAcrossLongRerouteChain(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 10000, Rows: 10000, TileSize: 16})
	runner := NewRunner(geom.Point{
		X: (4990.0 + 0.5) * gameWorld.TileSize(),
		Y: (5000.0 + 0.5) * gameWorld.TileSize(),
	}, false, 0)
	other := NewRunner(geom.Point{
		X: (5008.0 + 0.5) * gameWorld.TileSize(),
		Y: (5003.0 + 0.5) * gameWorld.TileSize(),
	}, false, 0)
	m := newTestManager(gameWorld, runner, other)

	mustIssueMove := func(unitID int64, target geom.Point) {
		t.Helper()
		if err := m.IssueMoveOrder(unitID, target); err != nil {
			t.Fatalf("IssueMoveOrder(unit=%d, target=%+v) error = %v", unitID, target, err)
		}
	}
	mustIssueFire := func(unitID int64, direction geom.Point) {
		t.Helper()
		if err := m.IssueFireOrder(unitID, direction); err != nil {
			t.Fatalf("IssueFireOrder(unit=%d, direction=%+v) error = %v", unitID, direction, err)
		}
	}

	// These two initial orders mirror the setup in the gameplay log, including the nearby second
	// runner whose reserved path keeps the occupancy context realistic for later reroute requests.
	mustIssueMove(runner.UnitID(), geom.Point{X: 79994.0, Y: 80032.3})
	mustIssueMove(other.UnitID(), geom.Point{X: 80136.0, Y: 79976.0})

	afterUpdate := map[int64]tickOrderExpectation{
		1:  {sleepTime: 20, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		2:  {sleepTime: 19, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		3:  {sleepTime: 18, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		4:  {sleepTime: 17, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		5:  {sleepTime: 16, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		6:  {sleepTime: 15, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		7:  {sleepTime: 14, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		8:  {sleepTime: 13, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		9:  {sleepTime: 12, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		10: {sleepTime: 11, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		11: {sleepTime: 10, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 3, queuedKind: OrderKindFire, queuedPathLen: 0, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		12: {sleepTime: 9, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 4, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 79992.0, Y: 79992.0}, queuedPathLen: 8, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		13: {sleepTime: 8, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 4, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 79992.0, Y: 79992.0}, queuedPathLen: 8, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		14: {sleepTime: 7, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 4, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 79992.0, Y: 79992.0}, queuedPathLen: 8, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		15: {sleepTime: 6, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 4, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 79992.0, Y: 79992.0}, queuedPathLen: 8, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		16: {sleepTime: 5, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 4, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 79992.0, Y: 79992.0}, queuedPathLen: 8, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		17: {sleepTime: 4, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 5, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80024.0, Y: 79960.0}, queuedPathLen: 10, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		18: {sleepTime: 3, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 5, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80024.0, Y: 79960.0}, queuedPathLen: 10, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		19: {sleepTime: 2, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 5, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80024.0, Y: 79960.0}, queuedPathLen: 10, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		20: {sleepTime: 1, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 6, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedPathLen: 12, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		21: {sleepTime: 20, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		22: {sleepTime: 19, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		23: {sleepTime: 18, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 7, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80088.0, Y: 79896.0}, queuedPathLen: 13, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		24: {sleepTime: 17, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 7, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80088.0, Y: 79896.0}, queuedPathLen: 13, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		25: {sleepTime: 16, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 7, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80088.0, Y: 79896.0}, queuedPathLen: 13, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		26: {sleepTime: 15, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 8, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80136.0, Y: 79896.0}, queuedPathLen: 16, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		27: {sleepTime: 14, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 9, queuedKind: OrderKindFire, queuedPathLen: 0, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		28: {sleepTime: 13, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 10, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80136.0, Y: 79896.0}, queuedPathLen: 16, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		29: {sleepTime: 12, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 10, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80136.0, Y: 79896.0}, queuedPathLen: 16, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		30: {sleepTime: 11, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 10, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80136.0, Y: 79896.0}, queuedPathLen: 16, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		31: {sleepTime: 10, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 10, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80136.0, Y: 79896.0}, queuedPathLen: 16, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		32: {sleepTime: 9, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 11, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80184.0, Y: 79896.0}, queuedPathLen: 19, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		33: {sleepTime: 8, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 11, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80184.0, Y: 79896.0}, queuedPathLen: 19, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		34: {sleepTime: 7, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 11, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80184.0, Y: 79896.0}, queuedPathLen: 19, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		35: {sleepTime: 6, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 11, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80184.0, Y: 79896.0}, queuedPathLen: 19, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
	}

	afterAction := map[int64]tickOrderExpectation{
		10: {sleepTime: 11, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 3, queuedKind: OrderKindFire, queuedPathLen: 0, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		11: {sleepTime: 10, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 4, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 79992.0, Y: 79992.0}, queuedPathLen: 8, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		16: {sleepTime: 5, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 5, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80024.0, Y: 79960.0}, queuedPathLen: 10, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		19: {sleepTime: 2, pathLen: 8, activeOrderID: 1, activeKind: OrderKindMove, activeTarget: geom.Point{X: 79992.0, Y: 80040.0}, queuedOrderID: 6, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedPathLen: 12, reachedTileX: 4990, reachedTileY: 5000, lastUpdateTick: 1},
		22: {sleepTime: 19, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 7, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80088.0, Y: 79896.0}, queuedPathLen: 13, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		25: {sleepTime: 16, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 8, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80136.0, Y: 79896.0}, queuedPathLen: 16, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		26: {sleepTime: 15, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 9, queuedKind: OrderKindFire, queuedPathLen: 0, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		27: {sleepTime: 14, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 10, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80136.0, Y: 79896.0}, queuedPathLen: 16, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		31: {sleepTime: 10, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 11, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80184.0, Y: 79896.0}, queuedPathLen: 19, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
		35: {sleepTime: 6, pathLen: 11, activeOrderID: 6, activeKind: OrderKindMove, activeTarget: geom.Point{X: 80056.0, Y: 79928.0}, queuedOrderID: 12, queuedKind: OrderKindMove, queuedTarget: geom.Point{X: 80232.0, Y: 79912.0}, queuedPathLen: 22, reachedTileX: 4991, reachedTileY: 5000, lastUpdateTick: 21},
	}

	actions := map[int64]func(){
		10: func() { mustIssueFire(runner.UnitID(), geom.Point{X: 0.994, Y: 0.110}) },
		11: func() { mustIssueMove(runner.UnitID(), geom.Point{X: 79998.2, Y: 79992.4}) },
		16: func() { mustIssueMove(runner.UnitID(), geom.Point{X: 80023.9, Y: 79953.7}) },
		19: func() { mustIssueMove(runner.UnitID(), geom.Point{X: 80057.9, Y: 79925.8}) },
		22: func() { mustIssueMove(runner.UnitID(), geom.Point{X: 80091.4, Y: 79894.9}) },
		25: func() { mustIssueMove(runner.UnitID(), geom.Point{X: 80136.0, Y: 79888.2}) },
		26: func() { mustIssueFire(runner.UnitID(), geom.Point{X: 0.316, Y: 0.949}) },
		27: func() { mustIssueMove(runner.UnitID(), geom.Point{X: 80136.0, Y: 79888.2}) },
		31: func() { mustIssueMove(runner.UnitID(), geom.Point{X: 80184.0, Y: 79896.0}) },
		35: func() { mustIssueMove(runner.UnitID(), geom.Point{X: 80227.1, Y: 79918.6}) },
	}

	for tick := int64(1); tick <= 35; tick++ {
		m.Update(tick)
		assertTickOrderExpectation(t, runner, gameWorld.TileSize(), afterUpdate[tick], tick, "after update")

		action, hasAction := actions[tick]
		if !hasAction {
			continue
		}

		action()
		assertTickOrderExpectation(t, runner, gameWorld.TileSize(), afterAction[tick], tick, "after action")
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

	advanceFireOrderUntilProjectileSpawned(t, m, 1)
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

// assertTickOrderExpectation compares the internal movement/order runtime state against one
// expected snapshot for a concrete tick. The helper stays intentionally strict so the long
// reroute integration test can catch even subtle premature promotions from queued to active.
func assertTickOrderExpectation(t *testing.T, runner *NonStaticUnit, tileSize float64, expectation tickOrderExpectation, tick int64, stage string) {
	t.Helper()

	if runner.sleepTime != expectation.sleepTime {
		t.Fatalf("%s tick %d sleepTime = %d, want %d", stage, tick, runner.sleepTime, expectation.sleepTime)
	}
	if len(runner.path) != expectation.pathLen {
		t.Fatalf("%s tick %d path len = %d, want %d", stage, tick, len(runner.path), expectation.pathLen)
	}
	if !runner.activeOrder.hasOrder {
		t.Fatalf("%s tick %d activeOrder.hasOrder = false, want true", stage, tick)
	}
	if runner.activeOrder.order.id != expectation.activeOrderID {
		t.Fatalf("%s tick %d activeOrder.id = %d, want %d", stage, tick, runner.activeOrder.order.id, expectation.activeOrderID)
	}
	if runner.activeOrder.order.kind != expectation.activeKind {
		t.Fatalf("%s tick %d activeOrder.kind = %v, want %v", stage, tick, runner.activeOrder.order.kind, expectation.activeKind)
	}
	if runner.activeOrder.order.targetPoint != expectation.activeTarget {
		t.Fatalf("%s tick %d activeOrder.target = %+v, want %+v", stage, tick, runner.activeOrder.order.targetPoint, expectation.activeTarget)
	}
	if expectation.queuedOrderID == 0 {
		if runner.queuedOrder.hasOrder {
			t.Fatalf("%s tick %d queuedOrder = %+v, want no queued order", stage, tick, runner.queuedOrder.order)
		}
	} else {
		if !runner.queuedOrder.hasOrder {
			t.Fatalf("%s tick %d queuedOrder.hasOrder = false, want true", stage, tick)
		}
		if runner.queuedOrder.order.id != expectation.queuedOrderID {
			t.Fatalf("%s tick %d queuedOrder.id = %d, want %d", stage, tick, runner.queuedOrder.order.id, expectation.queuedOrderID)
		}
		if runner.queuedOrder.order.kind != expectation.queuedKind {
			t.Fatalf("%s tick %d queuedOrder.kind = %v, want %v", stage, tick, runner.queuedOrder.order.kind, expectation.queuedKind)
		}
		if runner.queuedOrder.order.targetPoint != expectation.queuedTarget {
			t.Fatalf("%s tick %d queuedOrder.target = %+v, want %+v", stage, tick, runner.queuedOrder.order.targetPoint, expectation.queuedTarget)
		}
		if len(runner.queuedOrder.order.path) != expectation.queuedPathLen {
			t.Fatalf("%s tick %d queuedOrder.path len = %d, want %d", stage, tick, len(runner.queuedOrder.order.path), expectation.queuedPathLen)
		}
	}

	reachedTileX, reachedTileY := runner.ReachedTilePosition(tileSize)
	if reachedTileX != expectation.reachedTileX || reachedTileY != expectation.reachedTileY {
		t.Fatalf(
			"%s tick %d reached tile = (%d, %d), want (%d, %d)",
			stage,
			tick,
			reachedTileX,
			reachedTileY,
			expectation.reachedTileX,
			expectation.reachedTileY,
		)
	}
	if runner.lastUpdateTick != expectation.lastUpdateTick {
		t.Fatalf("%s tick %d lastUpdateTick = %d, want %d", stage, tick, runner.lastUpdateTick, expectation.lastUpdateTick)
	}
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

	explosionTicks := impactDurationTicks + 1
	for tick := int64(0); tick < int64(explosionTicks); tick++ {
		m.Update(100 + tick)
	}
}

func advanceFireOrderUntilProjectileSpawned(t *testing.T, m *Manager, startTick int64) int64 {
	t.Helper()

	for tick := startTick; tick <= startTick+fireOrderWindupTicks+2; tick++ {
		m.Update(tick)
		if projectileCount(m) > 0 {
			return tick
		}
	}

	t.Fatal("expected projectile to spawn after fire-order windup")
	return 0
}

func containsOrderStatus(reports []OrderReport, status OrderStatus) bool {
	for _, report := range reports {
		if report.Status == status {
			return true
		}
	}

	return false
}

func assertOrderStatusesPresent(t *testing.T, reports []OrderReport, statuses ...OrderStatus) {
	t.Helper()

	for _, status := range statuses {
		if containsOrderStatus(reports, status) {
			continue
		}

		t.Fatalf("order reports %+v do not contain status %v", reports, status)
	}
}

func reportsForOrderID(reports []OrderReport, orderID int64) []OrderReport {
	filtered := make([]OrderReport, 0, len(reports))
	for _, report := range reports {
		if report.OrderID != orderID {
			continue
		}
		filtered = append(filtered, report)
	}

	return filtered
}

func reportsForTargetPoint(reports []OrderReport, targetPoint geom.Point) []OrderReport {
	filtered := make([]OrderReport, 0, len(reports))
	for _, report := range reports {
		if report.TargetPoint != targetPoint {
			continue
		}
		filtered = append(filtered, report)
	}

	return filtered
}

func TestManagerDrainCombatEventsReportsProjectileSpawnHitAndKill(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	shooter := NewRunner(geom.Point{X: 24, Y: 24}, false, 0)
	target := NewRunner(geom.Point{X: 37, Y: 28}, false, 0)
	target.Health = 1
	m := newTestManager(gameWorld, shooter, target)

	if err := m.IssueFireOrder(shooter.UnitID(), geom.Point{X: 1, Y: 0}); err != nil {
		t.Fatalf("IssueFireOrder() error = %v", err)
	}

	collected := make([]CombatEvent, 0)
	for tick := int64(1); tick <= 40; tick++ {
		m.Update(tick)
		collected = append(collected, m.DrainCombatEvents()...)
		if containsCombatEventType(collected, CombatEventUnitKilled) {
			break
		}
	}

	assertCombatEventTypesPresent(t, collected, CombatEventProjectileSpawned, CombatEventProjectileHit, CombatEventUnitKilled)
	if containsCombatEventType(collected, CombatEventProjectileExpired) {
		t.Fatal("unexpected projectile_expired event for a projectile that hit and killed its target")
	}
}

func TestManagerDuelSnapshotReportsQueuedFireOrderAndCooldown(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	shooter := NewRunner(geom.Point{X: 24, Y: 24}, false, 0)
	target := NewRunner(geom.Point{X: 56, Y: 24}, false, 0)
	m := newTestManager(gameWorld, shooter, target)

	if err := m.IssueFireOrder(shooter.UnitID(), geom.Point{X: 1, Y: 0}); err != nil {
		t.Fatalf("IssueFireOrder() error = %v", err)
	}

	queuedSnapshot, ok := m.DuelSnapshot(shooter.UnitID(), target.UnitID())
	if !ok {
		t.Fatal("DuelSnapshot() = false, want true before the first update")
	}
	if !queuedSnapshot.Shooter.HasQueuedFireOrder {
		t.Fatal("expected queued fire order to be visible before execution starts")
	}

	spawnTick := advanceFireOrderUntilProjectileSpawned(t, m, 1)
	cooldownSnapshot, ok := m.DuelSnapshot(shooter.UnitID(), target.UnitID())
	if !ok {
		t.Fatal("DuelSnapshot() = false, want true after projectile spawn")
	}
	if cooldownSnapshot.Tick != spawnTick {
		t.Fatalf("snapshot tick = %d, want %d", cooldownSnapshot.Tick, spawnTick)
	}
	if cooldownSnapshot.Shooter.WeaponReady {
		t.Fatal("expected weapon to be cooling down immediately after fire-order completion")
	}
	if cooldownSnapshot.Shooter.FireCooldownRemaining == 0 {
		t.Fatal("expected cooldown counter to be visible in duel snapshot")
	}
}

func firstOrderedUnitID(t *testing.T, units *orderedUnitMap) int64 {
	t.Helper()

	for slotIndex := 0; slotIndex < units.SlotsLen(); slotIndex++ {
		first, ok := units.At(slotIndex)
		if !ok || first == nil {
			continue
		}

		return first.UnitID()
	}

	t.Fatal("expected at least one unit in ordered manager storage")
	return 0
}

func newTestManager(gameWorld world.World, units ...Unit) *Manager {
	manager := NewManager(gameWorld)
	for _, current := range units {
		manager.AddUnit(current)
	}
	return manager
}

func containsCombatEventType(events []CombatEvent, eventType CombatEventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}

	return false
}

func assertCombatEventTypesPresent(t *testing.T, events []CombatEvent, eventTypes ...CombatEventType) {
	t.Helper()

	for _, eventType := range eventTypes {
		if containsCombatEventType(events, eventType) {
			continue
		}

		t.Fatalf("combat events %+v do not contain event type %q", events, eventType)
	}
}
