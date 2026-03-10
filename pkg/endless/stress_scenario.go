package endless

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

const (
	stressStaticObjectCount      = 100000
	stressUnitCount              = 1000
	stressUnitSpawnIntervalTicks = 1
	stressSpawnSafeRadius        = 60
	stressJobTargetRadius        = 280
	stressJobRetryLimit          = 64
	stressSpawnColumns           = 40
	stressSpawnRows              = 25
	stressSpawnSpacingTiles      = 2
	stressActorID                = 1
	stressStaticSeed             = 2
	stressSeedLogInterval        = 10000
)

type stressScenario struct {
	actor              *stressActor
	pendingSpawnPoints []geom.Point
	nextSpawnTick      int64
	spawnedUnits       int
	staticObjects      int
}

// newStressScenario prepares the heavy-load scene requested for manual profiling. Static
// blockers are still planned up front, but they are now injected through Manager.AddUnit so the
// manager boot path never bypasses its normal registration logic.
func newStressScenario(gameWorld world.World) *stressScenario {
	startedAt := time.Now()
	centerTileX := gameWorld.Columns() / 2
	centerTileY := gameWorld.Rows() / 2
	blockedTiles := make(map[int64]struct{}, stressStaticObjectCount)
	log.Printf("[startup] stress: creating scenario for center tile (%d, %d)", centerTileX, centerTileY)

	spawnPointsStartedAt := time.Now()
	spawnPoints := buildStressSpawnPoints(gameWorld, centerTileX, centerTileY)
	log.Printf("[startup] stress: prepared %d spawn points in %s", len(spawnPoints), time.Since(spawnPointsStartedAt))

	scenario := &stressScenario{
		actor:              newStressActor(gameWorld, centerTileX, centerTileY, blockedTiles),
		pendingSpawnPoints: spawnPoints,
		nextSpawnTick:      spawnIntervalTicks(),
		spawnedUnits:       0,
		staticObjects:      stressStaticObjectCount,
	}
	log.Printf("[startup] stress: scenario struct ready in %s", time.Since(startedAt))
	return scenario
}

// SeedUnits creates the stress harness obstacle field through the public manager API so the
// manager constructor can stay empty and every static body still gets normal tile-stack
// registration and ID assignment.
func (s *stressScenario) SeedUnits(manager *unit.Manager) {
	if s == nil || manager == nil {
		return
	}

	startedAt := time.Now()
	log.Printf("[startup] stress: static unit seeding started (%d objects planned)", stressStaticObjectCount)

	buildStartedAt := time.Now()
	staticUnits := buildStressStaticUnits(s.actor.world, s.actor.centerTileX, s.actor.centerTileY, s.actor.blocked)
	log.Printf("[startup] stress: built %d static units in %s", len(staticUnits), time.Since(buildStartedAt))

	registerStartedAt := time.Now()
	for index, current := range staticUnits {
		manager.AddUnit(current)
		if (index+1)%stressSeedLogInterval == 0 || index+1 == len(staticUnits) {
			log.Printf(
				"[startup] stress: registered %d/%d static units in %s (seed total %s)",
				index+1,
				len(staticUnits),
				time.Since(registerStartedAt),
				time.Since(startedAt),
			)
		}
	}

	log.Printf("[startup] stress: static unit seeding finished in %s", time.Since(startedAt))
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

// DebugText exposes one compact scene summary for the in-game overlay so the dedicated stress
// launcher can confirm that the expected heavy-load harness was actually seeded.
func (s *stressScenario) DebugText() string {
	if s == nil {
		return ""
	}

	return fmt.Sprintf(
		"Scene: stress  units %d/%d  static objects %d  jobs completed %d  jobs failed %d",
		s.SpawnedUnits(),
		stressUnitCount,
		s.StaticObjects(),
		s.JobCompletedCount(),
		s.JobFailedCount(),
	)
}

// spawnReadyUnits releases runners one by one using a fixed tick cadence. The actor starts
// managing each unit immediately so the new runner receives its first move job before the
// simulation step of the same tick.
func (s *stressScenario) spawnReadyUnits(gameTick int64, manager *unit.Manager) {
	for s.spawnedUnits < len(s.pendingSpawnPoints) && gameTick >= s.nextSpawnTick {
		spawnIndex := s.spawnedUnits
		spawnPos := s.pendingSpawnPoints[spawnIndex]
		runnerID := manager.AddUnit(unit.NewRunner(
			spawnPos,
			spawnIndex%2 == 1,
			(spawnIndex%8)*3,
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

// Update drains reports only for the units this actor owns and immediately assigns a new move
// job to every managed unit that is currently without one. This keeps runners moving
// continuously without forcing the manager to rescan unrelated units for actor-facing events.
func (a *stressActor) Update(manager *unit.Manager) {
	if a == nil || manager == nil {
		return
	}

	for unitID := range a.managed {
		for _, report := range manager.DrainUnitJobReports(unitID) {
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
	startedAt := time.Now()
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
		} else {
			staticUnits = append(staticUnits, unit.NewBarricade(position))
		}
		if len(staticUnits)%stressSeedLogInterval == 0 || len(staticUnits) == stressStaticObjectCount {
			log.Printf("[startup] stress: generated %d/%d static units in %s", len(staticUnits), stressStaticObjectCount, time.Since(startedAt))
		}
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
	return stressUnitSpawnIntervalTicks
}

func packTile(tileX, tileY int) int64 {
	return int64(uint32(tileX))<<32 | int64(uint32(tileY))
}
