package endless

import (
	"math"
	"math/rand"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

const (
	stressStaticObjectCount   = 100000
	stressUnitCount           = 1000
	stressUnitSpawnIntervalMs = 100
	stressSpawnSafeRadius     = 60
	stressJobTargetRadius     = 280
	stressJobRetryLimit       = 64
	stressSpawnColumns        = 40
	stressSpawnRows           = 25
	stressSpawnSpacingTiles   = 2
	stressActorID             = 1
	stressStaticSeed          = 2
)

type stressScenario struct {
	actor              *stressActor
	pendingSpawnPoints []geom.Point
	nextSpawnTick      int64
	spawnedUnits       int
	staticObjects      int
}

// newStressScenario prepares the heavy-load scene requested for manual profiling: a large
// amount of static blockers is created immediately while mobile runners are deferred so the
// user can watch the system ramp up over time instead of paying one huge spawn spike.
func newStressScenario(gameWorld world.World) ([]unit.Unit, *stressScenario) {
	centerTileX := gameWorld.Columns() / 2
	centerTileY := gameWorld.Rows() / 2
	blockedTiles := make(map[int64]struct{}, stressStaticObjectCount)
	staticUnits := buildStressStaticUnits(gameWorld, centerTileX, centerTileY, blockedTiles)

	return staticUnits, &stressScenario{
		actor:              newStressActor(gameWorld, centerTileX, centerTileY, blockedTiles),
		pendingSpawnPoints: buildStressSpawnPoints(gameWorld, centerTileX, centerTileY),
		nextSpawnTick:      spawnIntervalTicks(),
		spawnedUnits:       0,
		staticObjects:      len(staticUnits),
	}
}

// Update advances the scenario-specific orchestration before the main unit simulation step.
// First it spawns any runner whose delay has expired, then it lets the actor react to job
// reports and assign the next move order to units that have gone idle.
func (s *stressScenario) Update(gameTick int64, manager *unit.Manager) {
	if s == nil || manager == nil {
		return
	}

	s.spawnReadyUnits(gameTick, manager)
	s.actor.Update(manager)
}

func (s *stressScenario) SpawnedUnits() int {
	if s == nil {
		return 0
	}

	return s.spawnedUnits
}

func (s *stressScenario) StaticObjects() int {
	if s == nil {
		return 0
	}

	return s.staticObjects
}

func (s *stressScenario) JobCompletedCount() int64 {
	if s == nil || s.actor == nil {
		return 0
	}

	return s.actor.completedJobs
}

func (s *stressScenario) JobFailedCount() int64 {
	if s == nil || s.actor == nil {
		return 0
	}

	return s.actor.failedJobs
}

// spawnReadyUnits releases runners one by one using the requested 100 ms cadence. The actor
// starts managing each unit immediately so the new runner receives its first move job before
// the simulation step of the same tick.
func (s *stressScenario) spawnReadyUnits(gameTick int64, manager *unit.Manager) {
	for s.spawnedUnits < len(s.pendingSpawnPoints) && gameTick >= s.nextSpawnTick {
		spawnIndex := s.spawnedUnits
		spawnPos := s.pendingSpawnPoints[spawnIndex]
		runnerID := manager.AddUnit(unit.NewRunner(
			spawnPos,
			spawnIndex%2 == 1,
			float64(spawnIndex%8)*0.05,
		))
		s.actor.RegisterUnit(runnerID)
		s.spawnedUnits++
		s.nextSpawnTick += spawnIntervalTicks()
	}
}

type stressActor struct {
	id          int64
	world       world.World
	rng         *rand.Rand
	centerTileX int
	centerTileY int
	blocked     map[int64]struct{}
	managed     map[int64]struct{}
	inFlight    map[int64]int64
	nextJobID   int64

	completedJobs int64
	failedJobs    int64
}

// newStressActor creates the single job-owning actor used by the stress harness. The actor
// keeps only the state required to reissue movement jobs after each completion or failure so
// the hot loop remains easy to inspect during profiling.
func newStressActor(gameWorld world.World, centerTileX, centerTileY int, blocked map[int64]struct{}) *stressActor {
	return &stressActor{
		id:          stressActorID,
		world:       gameWorld,
		rng:         rand.New(rand.NewSource(1)),
		centerTileX: centerTileX,
		centerTileY: centerTileY,
		blocked:     blocked,
		managed:     make(map[int64]struct{}, stressUnitCount),
		inFlight:    make(map[int64]int64, stressUnitCount),
	}
}

func (a *stressActor) RegisterUnit(unitID int64) {
	if a == nil || unitID == 0 {
		return
	}

	a.managed[unitID] = struct{}{}
}

// Update drains unit reports from the manager and immediately assigns a new move job to every
// managed unit that is currently without one. This keeps runners moving continuously while
// still routing all ownership through explicit job completion events.
func (a *stressActor) Update(manager *unit.Manager) {
	if a == nil || manager == nil {
		return
	}

	for _, report := range manager.DrainJobReports() {
		if report.ActorID != a.id {
			continue
		}

		delete(a.inFlight, report.UnitID)
		if report.Status == unit.JobStatusCompleted {
			a.completedJobs++
			continue
		}

		a.failedJobs++
	}

	for unitID := range a.managed {
		if _, busy := a.inFlight[unitID]; busy {
			continue
		}

		job := a.nextMoveJob(unitID)
		if err := manager.AssignMoveJob(job); err != nil {
			continue
		}

		a.inFlight[unitID] = job.ID
	}
}

func (a *stressActor) nextMoveJob(unitID int64) unit.MoveJob {
	targetTileX, targetTileY := a.randomOpenTargetTile()
	a.nextJobID++

	return unit.MoveJob{
		ID:          a.nextJobID,
		ActorID:     a.id,
		UnitID:      unitID,
		TargetTileX: targetTileX,
		TargetTileY: targetTileY,
	}
}

// randomOpenTargetTile samples the stress arena until it finds a tile that is inside the
// world and not reserved by the static-object layout. Keeping this check here avoids wasting
// actor ticks on obviously invalid destinations.
func (a *stressActor) randomOpenTargetTile() (int, int) {
	for attempt := 0; attempt < stressJobRetryLimit; attempt++ {
		tileX := a.centerTileX + a.rng.Intn(stressJobTargetRadius*2+1) - stressJobTargetRadius
		tileY := a.centerTileY + a.rng.Intn(stressJobTargetRadius*2+1) - stressJobTargetRadius
		if !a.world.InBounds(tileX, tileY) {
			continue
		}
		if _, blocked := a.blocked[packTile(tileX, tileY)]; blocked {
			continue
		}

		return tileX, tileY
	}

	return a.centerTileX, a.centerTileY
}

// buildStressStaticUnits creates the requested 100 000 static blockers by sampling unique
// random tiles across the whole map. The center spawn arena stays clear so delayed runner
// spawning and the first actor jobs always begin from an obstacle-free patch.
func buildStressStaticUnits(gameWorld world.World, centerTileX, centerTileY int, blocked map[int64]struct{}) []unit.Unit {
	staticUnits := make([]unit.Unit, 0, stressStaticObjectCount)
	rng := rand.New(rand.NewSource(stressStaticSeed))

	for len(staticUnits) < stressStaticObjectCount {
		tileX := rng.Intn(gameWorld.Columns())
		tileY := rng.Intn(gameWorld.Rows())
		if math.Abs(float64(tileX-centerTileX)) <= stressSpawnSafeRadius &&
			math.Abs(float64(tileY-centerTileY)) <= stressSpawnSafeRadius {
			continue
		}
		key := packTile(tileX, tileY)
		if _, occupied := blocked[key]; occupied {
			continue
		}

		blocked[key] = struct{}{}
		position := cellAnchor(tileX, tileY, gameWorld.TileSize())
		if rng.Intn(2) == 0 {
			staticUnits = append(staticUnits, unit.NewWall(position))
			continue
		}

		staticUnits = append(staticUnits, unit.NewBarricade(position))
	}

	return staticUnits
}

// buildStressSpawnPoints prepares exactly 1 000 runner anchors inside the obstacle-free arena
// at the map center. The evenly spaced grid keeps the initial formation deterministic and
// avoids units spawning on top of one another.
func buildStressSpawnPoints(gameWorld world.World, centerTileX, centerTileY int) []geom.Point {
	spawnPoints := make([]geom.Point, 0, stressUnitCount)
	startTileX := centerTileX - ((stressSpawnColumns - 1) * stressSpawnSpacingTiles / 2)
	startTileY := centerTileY - ((stressSpawnRows - 1) * stressSpawnSpacingTiles / 2)

	for row := 0; row < stressSpawnRows; row++ {
		for col := 0; col < stressSpawnColumns; col++ {
			tileX := startTileX + col*stressSpawnSpacingTiles
			tileY := startTileY + row*stressSpawnSpacingTiles
			if !gameWorld.InBounds(tileX, tileY) {
				continue
			}

			spawnPoints = append(spawnPoints, cellAnchor(tileX, tileY, gameWorld.TileSize()))
			if len(spawnPoints) == stressUnitCount {
				return spawnPoints
			}
		}
	}

	return spawnPoints
}

func spawnIntervalTicks() int64 {
	return int64(math.Round(stressUnitSpawnIntervalMs / (1000.0 / tps)))
}

func packTile(tileX, tileY int) int64 {
	return int64(uint32(tileX))<<32 | int64(uint32(tileY))
}
