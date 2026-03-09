package unit

import (
	"testing"

	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/world"
)

func TestManagerPanelRectHiddenWhenSelectedUnitOffScreen(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 16, Y: 16}, false, 0),
	})
	cam := camera.New(camera.Config{})

	m.SyncVisibility(cam, 64, 64)
	m.SelectAtScreen(cam, geom.Point{X: 16, Y: 16}, 64, 64)

	if _, ok := m.PanelRect(64, 64); !ok {
		t.Fatal("expected panel rect for visible selected unit")
	}

	cam.SetPosition(geom.Point{X: 128, Y: 128})
	m.SyncVisibility(cam, 64, 64)

	if _, ok := m.PanelRect(64, 64); ok {
		t.Fatal("expected panel rect to be hidden for offscreen selected unit")
	}
}

func TestManagerSelectAtScreenIgnoresOffScreenUnits(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 160, Y: 160}, false, 0),
	})
	cam := camera.New(camera.Config{})

	m.SyncVisibility(cam, 64, 64)
	m.SelectAtScreen(cam, geom.Point{X: 16, Y: 16}, 64, 64)

	if m.HasSelected() {
		t.Fatal("expected offscreen unit to be ignored by screen selection")
	}
}

func TestManagerSelectAtScreenUsesTileHitInsteadOfSpriteRect(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
	})
	cam := camera.New(camera.Config{})

	m.SyncVisibility(cam, 64, 64)
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
	m := NewManager(gameWorld, []Unit{
		NewWall(geom.Point{X: 24, Y: 24}),
	})
	cam := camera.New(camera.Config{})

	m.SyncVisibility(cam, 64, 64)
	m.SelectAtScreen(cam, geom.Point{X: 20, Y: 20}, 64, 64)

	if !m.HasSelected() {
		t.Fatal("expected click inside static unit tile to select it")
	}
}

func TestManagerProjectileHitsUnitOccupyingEnteredTile(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	target := NewRunner(geom.Point{X: 37, Y: 28}, false, 0)
	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
		target,
	})
	m.selected = 0

	if err := m.CommandSelectedFire(geom.Point{X: 120, Y: 24}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	initialHealth := target.Health
	m.Update(1, 1.0/60.0)

	if target.Health != initialHealth-1 {
		t.Fatalf("target health = %d, want %d after projectile enters occupied tile", target.Health, initialHealth-1)
	}
	if count := countProjectiles(m.units); count != 0 {
		t.Fatalf("projectiles = %d, want 0 after hit", count)
	}
}

func TestManagerProjectileExpiresAfterMaxRange(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 64, Rows: 64, TileSize: 16})
	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
	})
	m.selected = 0

	if err := m.CommandSelectedFire(geom.Point{X: 512, Y: 24}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	for range 60 {
		m.Update(1, 1.0/60.0)
	}

	if count := countProjectiles(m.units); count != 0 {
		t.Fatalf("projectiles = %d, want projectile removed after max range", count)
	}
}

func TestManagerProjectileRespawnsUnitAtSpawnPoint(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	target := NewRunner(geom.Point{X: 37, Y: 28}, false, 0)
	target.SpawnPosition = geom.Point{X: 104, Y: 104}
	target.Health = 1

	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
		target,
	})
	m.selected = 0

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
	if count := countProjectiles(m.units); count != 0 {
		t.Fatalf("projectiles = %d, want projectile consumed on hit", count)
	}
}

func TestManagerProjectileCanDamageStaticUnit(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	target := NewWall(geom.Point{X: 37, Y: 28})
	target.Health = 1

	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
		target,
	})
	m.selected = 0

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

func TestManagerSyncVisibilityUpdatesProjectileOnScreenState(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 64, Rows: 64, TileSize: 16})
	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 24, Y: 24}, false, 0),
	})
	m.selected = 0

	if err := m.CommandSelectedFire(geom.Point{X: 120, Y: 24}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	cam := camera.New(camera.Config{})
	m.SyncVisibility(cam, 64, 64)

	projectile := firstProjectile(m.units)
	if projectile == nil {
		t.Fatal("expected projectile to exist")
	}
	if !projectile.Base().OnScreen {
		t.Fatal("expected projectile to be marked as visible")
	}

	cam.SetPosition(geom.Point{X: 400, Y: 400})
	m.SyncVisibility(cam, 64, 64)

	if projectile.Base().OnScreen {
		t.Fatal("expected projectile to be marked as offscreen")
	}
}

func firstProjectile(units []Unit) *Projectile {
	for _, current := range units {
		if projectile, ok := current.(*Projectile); ok {
			return projectile
		}
	}
	return nil
}

func countProjectiles(units []Unit) int {
	count := 0
	for _, current := range units {
		if _, ok := current.(*Projectile); ok {
			count++
		}
	}
	return count
}
