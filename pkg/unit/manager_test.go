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
