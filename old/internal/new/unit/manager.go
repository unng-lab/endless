package unit

import (
	"errors"

	"github.com/unng-lab/endless/internal/new/camera"
	"github.com/unng-lab/endless/internal/new/pathfinding"
	"github.com/unng-lab/endless/internal/new/tilemap"
)

// Manager keeps track of all units in the scene and coordinates pathfinding.
type Manager struct {
	tiles     *tilemap.TileMap
	navigator *pathfinding.Navigator
	units     []*Unit
	tileSize  float64
}

// ErrNoPath is returned when a movement order cannot be satisfied.
var ErrNoPath = errors.New("no path available")

// NewManager creates a unit manager for the provided map and navigator.
func NewManager(m *tilemap.TileMap, nav *pathfinding.Navigator) *Manager {
	if nav == nil {
		nav = pathfinding.NewNavigator(m, 32)
	}
	return &Manager{tiles: m, navigator: nav, tileSize: m.TileSize()}
}

// Add spawns a unit under management.
func (m *Manager) Add(u *Unit) {
	if u != nil {
		m.units = append(m.units, u)
	}
}

// Units returns the list of managed units.
func (m *Manager) Units() []*Unit {
	return m.units
}

// Update ticks all units forward.
func (m *Manager) Update(delta float64) {
	for _, u := range m.units {
		u.Update(delta)
	}
}

// CommandMove orders a unit to move towards the provided tile coordinate.
func (m *Manager) CommandMove(u *Unit, target pathfinding.Point) error {
	if u == nil {
		return nil
	}

	start := m.worldToTile(u.Position)
	path, ok := m.navigator.FindPath(start, target)
	if !ok || len(path) == 0 {
		return ErrNoPath
	}

	world := make([]camera.Point, 0, len(path))
	for _, p := range path {
		world = append(world, m.tileToWorld(p))
	}
	u.SetPath(world)
	return nil
}

func (m *Manager) tileToWorld(p pathfinding.Point) camera.Point {
	size := m.tileSize
	return camera.Point{
		X: (float64(p.X) + 0.5) * size,
		Y: (float64(p.Y) + 0.5) * size,
	}
}

func (m *Manager) worldToTile(p camera.Point) pathfinding.Point {
	size := m.tileSize
	return pathfinding.Point{X: int(p.X / size), Y: int(p.Y / size)}
}
