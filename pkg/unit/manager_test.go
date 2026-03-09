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

func TestManagerProjectileHitsOnlyWhenPassingThroughUnitPoint(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	target := NewRunner(geom.Point{X: 48, Y: 24}, false, 0)
	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 16, Y: 16}, false, 0),
		target,
	})
	m.selected = 0

	if err := m.CommandSelectedFire(geom.Point{X: 80, Y: 16}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	initialHealth := m.units[1].Health
	m.Update(0.2)

	if m.units[1].Health != initialHealth {
		t.Fatalf("target health = %d, want %d when projectile misses unit point", m.units[1].Health, initialHealth)
	}
	if len(m.projectiles) == 0 {
		t.Fatal("expected projectile to remain active after missing unit point")
	}
}

func TestManagerProjectileExpiresAfterMaxRange(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 64, Rows: 64, TileSize: 16})
	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 16, Y: 16}, false, 0),
	})
	m.selected = 0

	if err := m.CommandSelectedFire(geom.Point{X: 512, Y: 16}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	for range 60 {
		m.Update(1.0 / 60.0)
	}

	if len(m.projectiles) != 0 {
		t.Fatalf("projectiles = %d, want projectile removed after max range", len(m.projectiles))
	}
}

func TestManagerProjectileRespawnsUnitAtSpawnPoint(t *testing.T) {
	gameWorld := world.New(world.Config{Columns: 32, Rows: 32, TileSize: 16})
	target := NewRunner(geom.Point{X: 48, Y: 16}, false, 0)
	target.SpawnPosition = geom.Point{X: 96, Y: 96}
	target.Health = 1

	m := NewManager(gameWorld, []Unit{
		NewRunner(geom.Point{X: 16, Y: 16}, false, 0),
		target,
	})
	m.selected = 0

	if err := m.CommandSelectedFire(geom.Point{X: 80, Y: 16}); err != nil {
		t.Fatalf("CommandSelectedFire() error = %v", err)
	}

	m.Update(0.2)

	if m.units[1].Position != target.SpawnPosition {
		t.Fatalf("target position = %+v, want respawn at %+v", m.units[1].Position, target.SpawnPosition)
	}
	if m.units[1].Health != m.units[1].MaxHealth {
		t.Fatalf("target health = %d, want full health %d after respawn", m.units[1].Health, m.units[1].MaxHealth)
	}
	if len(m.projectiles) != 0 {
		t.Fatalf("projectiles = %d, want projectile consumed on hit", len(m.projectiles))
	}
}
