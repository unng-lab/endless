package endless

import (
	"fmt"

	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

const (
	basicRunnerCount       = 2
	basicStaticObjectCount = 4
)

// basicScenario seeds only a handful of controllable bodies near the map center so the normal
// launcher remains interactive and debuggable without carrying the full profiling load.
type basicScenario struct {
	world         world.World
	centerTileX   int
	centerTileY   int
	spawnedUnits  int
	staticObjects int
}

// newBasicScenario records the center anchor used for the lightweight default scene. The
// actual unit creation is deferred until SeedUnits so the manager still owns all IDs and
// registration side effects.
func newBasicScenario(gameWorld world.World) *basicScenario {
	return &basicScenario{
		world:       gameWorld,
		centerTileX: gameWorld.Columns() / 2,
		centerTileY: gameWorld.Rows() / 2,
	}
}

// SeedUnits places two runners and four nearby obstacles around the camera start position.
// This gives the regular launcher something to select, move and shoot at without recreating
// the expensive 100k-obstacle stress harness.
func (s *basicScenario) SeedUnits(manager *unit.Manager) {
	if s == nil || manager == nil {
		return
	}

	for index, tile := range s.runnerTiles() {
		manager.AddUnit(unit.NewRunner(
			cellAnchor(tile.tileX, tile.tileY, s.world.TileSize()),
			index%2 == 1,
			index*6,
		))
		s.spawnedUnits++
	}

	for index, tile := range s.staticTiles() {
		position := cellAnchor(tile.tileX, tile.tileY, s.world.TileSize())
		if index%2 == 0 {
			manager.AddUnit(unit.NewWall(position))
		} else {
			manager.AddUnit(unit.NewBarricade(position))
		}
		s.staticObjects++
	}
}

// Update is intentionally empty for the basic launcher. The regular scene exists only to seed
// a small manual-play sandbox, so no autonomous actor needs to keep issuing commands.
func (s *basicScenario) Update(gameTick int64, manager *unit.Manager) {
}

// DebugText reports the lightweight scene inventory so the on-screen overlay still shows which
// launcher mode populated the world.
func (s *basicScenario) DebugText() string {
	if s == nil {
		return ""
	}

	return fmt.Sprintf("Scene: basic  units %d  static objects %d", s.spawnedUnits, s.staticObjects)
}

// basicTileAnchor names one tile coordinate pair used by the basic scenario seed layout.
// Keeping the tile coordinates together avoids passing loosely related ints between helpers.
type basicTileAnchor struct {
	tileX int
	tileY int
}

// runnerTiles returns a symmetric pair of mobile spawn tiles that leave open space between
// the units and the obstacle ring, making the first manual commands easy to issue.
func (s *basicScenario) runnerTiles() []basicTileAnchor {
	tiles := make([]basicTileAnchor, 0, basicRunnerCount)
	tiles = append(tiles,
		basicTileAnchor{tileX: s.centerTileX - 2, tileY: s.centerTileY},
		basicTileAnchor{tileX: s.centerTileX + 2, tileY: s.centerTileY},
	)
	return tiles
}

// staticTiles returns a small obstacle cross around the map center. The pattern demonstrates
// selection and projectile collisions while still leaving multiple open approach directions.
func (s *basicScenario) staticTiles() []basicTileAnchor {
	tiles := make([]basicTileAnchor, 0, basicStaticObjectCount)
	tiles = append(tiles,
		basicTileAnchor{tileX: s.centerTileX, tileY: s.centerTileY - 3},
		basicTileAnchor{tileX: s.centerTileX - 3, tileY: s.centerTileY},
		basicTileAnchor{tileX: s.centerTileX + 3, tileY: s.centerTileY},
		basicTileAnchor{tileX: s.centerTileX, tileY: s.centerTileY + 3},
	)
	return tiles
}
