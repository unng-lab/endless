package scenario

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

const (
	basicRunnerCount       = 4
	basicStaticObjectCount = 4
)

// basicScenario seeds only a handful of controllable bodies near the map center so the normal
// launcher remains interactive and debuggable without carrying the full profiling load.
type basicScenario struct {
	world         world.World
	centerTileX   int
	centerTileY   int
	rng           *rand.Rand
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
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SeedUnits places four runners into a compact square around the initial camera focus and
// then adds four walls sampled from a wider ring around that square. The random wall layout
// keeps the default sandbox visually varied while still preserving open tiles immediately
// around the spawned units for the first move commands.
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

	for _, tile := range s.staticTiles() {
		position := cellAnchor(tile.tileX, tile.tileY, s.world.TileSize())
		manager.AddUnit(unit.NewWall(position))
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

// runnerTiles returns a symmetric 2x2 formation centered on the camera start tile. Using a
// square instead of a line keeps all four starters visible near the screen center at launch.
func (s *basicScenario) runnerTiles() []basicTileAnchor {
	tiles := make([]basicTileAnchor, 0, basicRunnerCount)
	tiles = append(tiles,
		basicTileAnchor{tileX: s.centerTileX - 1, tileY: s.centerTileY - 1},
		basicTileAnchor{tileX: s.centerTileX, tileY: s.centerTileY - 1},
		basicTileAnchor{tileX: s.centerTileX - 1, tileY: s.centerTileY},
		basicTileAnchor{tileX: s.centerTileX, tileY: s.centerTileY},
	)
	return tiles
}

// staticTiles chooses a small wall set from a ring of candidate tiles around the spawn square.
// Shuffling the candidates produces a different obstacle order each launch without risking a
// wall directly on top of the initial four units.
func (s *basicScenario) staticTiles() []basicTileAnchor {
	candidates := []basicTileAnchor{
		{tileX: s.centerTileX - 3, tileY: s.centerTileY - 2},
		{tileX: s.centerTileX, tileY: s.centerTileY - 3},
		{tileX: s.centerTileX + 2, tileY: s.centerTileY - 2},
		{tileX: s.centerTileX + 3, tileY: s.centerTileY},
		{tileX: s.centerTileX + 2, tileY: s.centerTileY + 2},
		{tileX: s.centerTileX, tileY: s.centerTileY + 3},
		{tileX: s.centerTileX - 2, tileY: s.centerTileY + 2},
		{tileX: s.centerTileX - 3, tileY: s.centerTileY},
	}
	if s.rng == nil {
		s.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	s.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	return append([]basicTileAnchor(nil), candidates[:basicStaticObjectCount]...)
}
