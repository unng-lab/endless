package unit

import (
	"image"
	"log"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/pkg/assets"
	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/world"
)

const updateBatchSize = 16

type Manager struct {
	world    world.World
	renderer *Renderer

	units *orderedUnitMap

	// debugExternalAPILogging gates verbose audit logs for external movement and fire commands.
	// The flag stays on the manager so visual/debug launchers may opt in without affecting the
	// headless RL collection path that issues the same gameplay API at much higher frequency.
	debugExternalAPILogging bool

	bufferedOrderReports map[int64][]OrderReport
	combatEvents         []CombatEvent
	tileStacks           map[tileKey]*TileStack
	registeredTiles      map[int64]tileKey
	selectedID           int64
	nextID               int64
	nextOrderID          int64
	lastGameTick         int64

	workers         []chan int64
	updateWG        sync.WaitGroup
	orderReportsMu  sync.Mutex
	combatEventsMu  sync.Mutex
	tileRegistryMu  sync.RWMutex
	pendingSpawnsMu sync.Mutex
	pendingSpawns   []Unit
	closeOnce       sync.Once
}

// tileEntryReactiveUnit describes units whose side effects must run exactly at the moment the
// manager has already moved the unit into a new tile and updated tile-stack membership.
// Projectiles use this hook to resolve impacts against the occupants of the tile they entered.
type tileEntryReactiveUnit interface {
	ReactToEnteredTile(*Manager, *TileStack)
}

// visibleUpdatingUnit describes units that maintain additional draw-only state for visible
// interpolation or animation. The manager invokes this hook while iterating visible tile
// stacks so the draw list is built and refreshed in one pass over the visible tiles.
type visibleUpdatingUnit interface {
	UpdateVisible(int64)
}

type tileKey struct {
	x int
	y int
}

// NewManager creates an empty unit manager. Callers must register every gameplay body through
// AddUnit so tile stacks, persistent IDs and ordered storage are initialized through the same
// runtime path that will later be used for dynamic spawns.
func NewManager(gameWorld world.World) *Manager {
	startedAt := time.Now()
	m := &Manager{
		world:                gameWorld,
		renderer:             NewRenderer(),
		units:                newOrderedUnitMap(0),
		bufferedOrderReports: make(map[int64][]OrderReport),
		combatEvents:         make([]CombatEvent, 0),
		tileStacks:           make(map[tileKey]*TileStack),
		registeredTiles:      make(map[int64]tileKey),
	}
	log.Printf("[startup] units: manager core structures allocated in %s", time.Since(startedAt))

	workersStartedAt := time.Now()
	m.startWorkers()
	log.Printf("[startup] units: worker pool started in %s", time.Since(workersStartedAt))
	return m
}

// Draw renders every tile-registered world body in the same tile order as the visible terrain
// pass. Callers may disable the extra visible-unit refresh when they need the draw traversal to
// reuse the current interpolated state without advancing visible-only animation or smoothing.
func (m *Manager) Draw(screen *ebiten.Image, cam *camera.Camera, quality assets.Quality, visible image.Rectangle, updateVisibleUnits bool) error {
	for _, current := range m.visibleTileUnits(visible, updateVisibleUnits) {
		if err := m.renderer.DrawUnit(screen, cam, m.world.TileSize(), quality, current); err != nil {
			return err
		}
	}

	return nil
}

// AddUnit registers a freshly spawned unit in the manager and returns the persistent ID that
// the caller should use for later commands, selections or order ownership tracking.
func (m *Manager) AddUnit(body Unit) int64 {
	if body == nil {
		return 0
	}
	body.Base().ClearRemovalMark()

	if body.UnitID() == 0 {
		m.nextID++
		body.SetUnitID(m.nextID)
	} else if body.UnitID() > m.nextID {
		m.nextID = body.UnitID()
	}

	m.bindUnitRuntimeDependencies(body)
	m.units.Set(body)
	m.registerUnitInCurrentTile(body)

	return body.UnitID()
}
