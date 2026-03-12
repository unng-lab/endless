package rl

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

// DuelEnvironment adapts the existing unit manager into one deterministic 1v1 RL environment.
// The shooter is controlled through ApplyAction, while the target keeps the current scripted
// patrol so data collection may start immediately without a second learned policy.
type DuelEnvironment struct {
	config DuelRunConfig

	manager   *unit.Manager
	gameWorld world.World
	shooterID int64
	targetID  int64

	tick                     int64
	targetWaypoints          []geom.Point
	nextTargetWaypoint       int
	targetMoveInFlight       bool
	previousTargetPos        geom.Point
	hasPreviousTargetPos     bool
	recentShooterMoveFailure bool
	lastObservation          Observation
	hasLastObservation       bool
}

// NewDuelEnvironment prepares one episode-scoped environment wrapper around the current duel
// scenario configuration.
func NewDuelEnvironment(config DuelRunConfig) *DuelEnvironment {
	return &DuelEnvironment{
		config: normalizedDuelRunConfig(config),
	}
}

// Reset rebuilds the world, manager and spawn layout for one deterministic episode seed.
func (e *DuelEnvironment) Reset(seed int64) (Observation, error) {
	e.Close()

	e.gameWorld = world.New(world.Config{
		Columns:  e.config.WorldColumns,
		Rows:     e.config.WorldRows,
		TileSize: e.config.TileSize,
	})
	e.manager = unit.NewManager(e.gameWorld)
	e.tick = 0
	e.targetMoveInFlight = false
	e.nextTargetWaypoint = 0
	e.previousTargetPos = geom.Point{}
	e.hasPreviousTargetPos = false
	e.recentShooterMoveFailure = false
	e.lastObservation = Observation{}
	e.hasLastObservation = false

	rng := rand.New(rand.NewSource(seed))
	layout := buildDuelScenarioLayout(rng, e.config.Scenario, e.gameWorld)
	for _, staticUnit := range layout.StaticUnits {
		e.manager.AddUnit(staticUnit)
	}
	e.shooterID = e.manager.AddUnit(unit.NewRunner(layout.ShooterSpawn, false, 0))
	e.targetID = e.manager.AddUnit(unit.NewRunner(layout.TargetSpawn, true, 6))
	e.targetWaypoints = append([]geom.Point(nil), layout.TargetWaypoints...)

	observation, err := e.Observe()
	if err != nil {
		return Observation{}, err
	}
	e.lastObservation = observation
	e.hasLastObservation = true
	return observation, nil
}

// Observe returns the current duel snapshot plus the environment metadata that policies use
// for action generation.
func (e *DuelEnvironment) Observe() (Observation, error) {
	if e == nil || e.manager == nil {
		return Observation{}, fmt.Errorf("duel environment is not initialized")
	}

	snapshot, ok := e.manager.DuelSnapshot(e.shooterID, e.targetID)
	if !ok {
		return Observation{}, fmt.Errorf("duel snapshot is unavailable")
	}
	return buildObservation(
		e.gameWorld,
		snapshot,
		e.manager.ProjectileSnapshots(),
		e.manager.BlockingUnitSnapshots(),
		e.previousTargetPos,
		e.hasPreviousTargetPos,
		e.recentShooterMoveFailure,
	), nil
}

// ApplyAction maps one policy decision back into the manager order API. Returning the accepted
// flag separately lets dataset rows keep the original intent even when gameplay validation
// rejects the command and emits a failed order report.
func (e *DuelEnvironment) ApplyAction(action Action) (bool, error) {
	if e == nil || e.manager == nil {
		return false, fmt.Errorf("duel environment is not initialized")
	}

	switch action.Type {
	case "", ActionTypeNone:
		return true, nil
	case ActionTypeMove:
		if err := e.manager.IssueMoveOrder(e.shooterID, action.MoveTarget); err != nil {
			return false, err
		}
		return true, nil
	case ActionTypeFire:
		if err := e.manager.IssueFireOrder(e.shooterID, action.FireDirection); err != nil {
			return false, err
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported action type %q", action.Type)
	}
}

// Step advances the world by one gameplay tick, collects every sparse event channel and returns
// the resulting post-tick observation together with reward and terminal state.
func (e *DuelEnvironment) Step() (StepResult, error) {
	if e == nil || e.manager == nil {
		return StepResult{}, fmt.Errorf("duel environment is not initialized")
	}
	if !e.hasLastObservation {
		observation, err := e.Observe()
		if err != nil {
			return StepResult{}, err
		}
		e.lastObservation = observation
		e.hasLastObservation = true
	}
	before := e.lastObservation

	e.tick++
	e.issueTargetPatrolOrder()
	e.manager.Update(e.tick)

	createdAt := time.Now().UTC()
	shooterReports := e.manager.DrainUnitOrderReports(e.shooterID)
	targetReports := e.manager.DrainUnitOrderReports(e.targetID)
	combatEvents := e.manager.DrainCombatEvents()
	if targetMoveReportFinished(targetReports) {
		e.targetMoveInFlight = false
	}
	e.recentShooterMoveFailure = shooterMoveFailed(shooterReports)

	afterSnapshot := resolvePostTickSnapshot(
		e.manager,
		e.shooterID,
		e.targetID,
		e.lastObservation.Snapshot,
		combatEvents,
	)
	reward := rewardForTick(e.shooterID, e.targetID, combatEvents)
	done := !afterSnapshot.Shooter.Alive || !afterSnapshot.Target.Alive || e.tick >= e.config.MaxTicksPerEpisode
	outcome := "in_progress"
	switch {
	case !afterSnapshot.Target.Alive:
		outcome = "target_killed"
	case !afterSnapshot.Shooter.Alive:
		outcome = "shooter_killed"
	case e.tick >= e.config.MaxTicksPerEpisode:
		outcome = "timeout"
	}

	after := buildObservation(
		e.gameWorld,
		afterSnapshot,
		e.manager.ProjectileSnapshots(),
		e.manager.BlockingUnitSnapshots(),
		before.Snapshot.Target.Position,
		before.Snapshot.Target.Alive,
		e.recentShooterMoveFailure,
	)

	e.previousTargetPos = before.Snapshot.Target.Position
	e.hasPreviousTargetPos = before.Snapshot.Target.Alive
	e.lastObservation = after
	e.hasLastObservation = true

	return StepResult{
		After:          after,
		Reward:         reward,
		Done:           done,
		Outcome:        outcome,
		ShooterReports: shooterReports,
		TargetReports:  targetReports,
		CombatEvents:   combatEvents,
		CreatedAt:      createdAt,
	}, nil
}

// Close releases the underlying unit manager worker pool so repeated episode generation does
// not accumulate background goroutines.
func (e *DuelEnvironment) Close() {
	if e == nil || e.manager == nil {
		return
	}

	e.manager.Close()
	e.manager = nil
}

func (e *DuelEnvironment) issueTargetPatrolOrder() {
	if e == nil || e.manager == nil || e.targetMoveInFlight || len(e.targetWaypoints) == 0 {
		return
	}

	targetPoint := e.targetWaypoints[e.nextTargetWaypoint]
	if err := e.manager.IssueMoveOrder(e.targetID, targetPoint); err != nil {
		return
	}

	e.targetMoveInFlight = true
	e.nextTargetWaypoint = (e.nextTargetWaypoint + 1) % len(e.targetWaypoints)
}

func shooterMoveFailed(reports []unit.OrderReport) bool {
	for _, report := range reports {
		if report.Kind == unit.OrderKindMove && report.Status == unit.OrderFailed {
			return true
		}
	}
	return false
}
