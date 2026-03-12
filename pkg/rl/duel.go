package rl

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

const duelScenarioName = "duel_fire_collection"

// DuelRunConfig describes one headless data-generation session.
type DuelRunConfig struct {
	Episodes           int
	MaxTicksPerEpisode int64
	Seed               int64
	WorldColumns       int
	WorldRows          int
	TileSize           float64
}

// RunDuelCollection executes deterministic fire-focused duel episodes and streams their
// resulting step/event traces into the supplied recorder.
func RunDuelCollection(ctx context.Context, config DuelRunConfig, recorder Recorder) error {
	config = normalizedDuelRunConfig(config)
	if recorder == nil {
		recorder = NoopRecorder{}
	}

	sessionRNG := rand.New(rand.NewSource(config.Seed))
	for episodeIndex := 0; episodeIndex < config.Episodes; episodeIndex++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		episodeSeed := sessionRNG.Int63()
		episodeID := uint64(episodeIndex + 1)
		if err := runOneDuelEpisode(ctx, episodeID, episodeSeed, config, recorder); err != nil {
			return fmt.Errorf("run duel episode %d: %w", episodeID, err)
		}
	}

	return recorder.Close(ctx)
}

type duelEpisodeState struct {
	targetWaypoints      []geom.Point
	nextTargetWaypoint   int
	targetMoveInFlight   bool
	previousTargetPos    geom.Point
	hasPreviousTargetPos bool
}

func runOneDuelEpisode(ctx context.Context, episodeID uint64, episodeSeed int64, config DuelRunConfig, recorder Recorder) error {
	gameWorld := world.New(world.Config{
		Columns:  config.WorldColumns,
		Rows:     config.WorldRows,
		TileSize: config.TileSize,
	})
	manager := unit.NewManager(gameWorld)
	rng := rand.New(rand.NewSource(episodeSeed))
	startedAt := time.Now().UTC()

	shooterSpawn, targetSpawn, targetWaypoints := duelLayout(rng, gameWorld)
	shooterID := manager.AddUnit(unit.NewRunner(shooterSpawn, false, 0))
	targetID := manager.AddUnit(unit.NewRunner(targetSpawn, true, 6))

	state := duelEpisodeState{
		targetWaypoints:    targetWaypoints,
		nextTargetWaypoint: 0,
	}
	totalReward := float32(0)
	outcome := "timeout"
	ticksExecuted := uint32(0)

	for tick := int64(1); tick <= config.MaxTicksPerEpisode; tick++ {
		ticksExecuted = uint32(tick)
		before, ok := manager.DuelSnapshot(shooterID, targetID)
		if !ok {
			outcome = "setup_failed"
			break
		}

		actionType, actionDirection := decideShooterAction(before, state.previousTargetPos, state.hasPreviousTargetPos)
		if actionType == "fire" {
			if err := manager.IssueFireOrder(shooterID, actionDirection); err != nil {
				actionType = "none"
				actionDirection = geom.Point{}
			}
		}

		if !state.targetMoveInFlight && len(state.targetWaypoints) > 0 {
			targetPoint := state.targetWaypoints[state.nextTargetWaypoint]
			if err := manager.IssueMoveOrder(targetID, targetPoint); err == nil {
				state.targetMoveInFlight = true
				state.nextTargetWaypoint = (state.nextTargetWaypoint + 1) % len(state.targetWaypoints)
			}
		}

		manager.Update(tick)
		createdAt := time.Now().UTC()

		shooterReports := manager.DrainUnitOrderReports(shooterID)
		targetReports := manager.DrainUnitOrderReports(targetID)
		combatEvents := manager.DrainCombatEvents()

		if targetMoveReportFinished(targetReports) {
			state.targetMoveInFlight = false
		}

		eventRecords := append(
			orderReportsToEventRecords(episodeID, tick, shooterReports, createdAt),
			orderReportsToEventRecords(episodeID, tick, targetReports, createdAt)...,
		)
		eventRecords = append(eventRecords, combatEventsToEventRecords(episodeID, combatEvents, createdAt)...)
		if err := recorder.RecordEvents(ctx, eventRecords); err != nil {
			return err
		}

		after := resolvePostTickSnapshot(manager, shooterID, targetID, before, combatEvents)
		reward := rewardForTick(shooterID, targetID, combatEvents)
		totalReward += reward

		done := !after.Shooter.Alive || !after.Target.Alive || tick == config.MaxTicksPerEpisode
		if !after.Target.Alive {
			outcome = "target_killed"
		} else if !after.Shooter.Alive {
			outcome = "shooter_killed"
		}

		stepRecord := StepRecord{
			EpisodeID:                 episodeID,
			Tick:                      uint32(tick),
			ShooterID:                 after.Shooter.UnitID,
			TargetID:                  after.Target.UnitID,
			ShooterX:                  float32(after.Shooter.Position.X),
			ShooterY:                  float32(after.Shooter.Position.Y),
			ShooterHP:                 int16(after.Shooter.Health),
			TargetX:                   float32(after.Target.Position.X),
			TargetY:                   float32(after.Target.Position.Y),
			TargetHP:                  int16(after.Target.Health),
			RelativeTargetX:           float32(after.RelativeTarget.X),
			RelativeTargetY:           float32(after.RelativeTarget.Y),
			DistanceToTarget:          float32(after.DistanceToTarget),
			ProjectileCount:           uint16(after.ProjectileCount),
			ShooterWeaponReady:        boolToUInt8(after.Shooter.WeaponReady),
			ShooterCooldownRemaining:  uint16(maxInt(after.Shooter.FireCooldownRemaining, 0)),
			ShooterHasActiveFireOrder: boolToUInt8(after.Shooter.HasActiveFireOrder),
			ShooterHasQueuedFireOrder: boolToUInt8(after.Shooter.HasQueuedFireOrder),
			ActionType:                actionType,
			ActionDirX:                float32(actionDirection.X),
			ActionDirY:                float32(actionDirection.Y),
			Reward:                    reward,
			Done:                      boolToUInt8(done),
			CreatedAt:                 createdAt,
		}
		if err := recorder.RecordStep(ctx, stepRecord); err != nil {
			return err
		}

		state.previousTargetPos = after.Target.Position
		state.hasPreviousTargetPos = after.Target.Alive
		if done {
			break
		}
	}

	return recorder.RecordEpisode(ctx, EpisodeRecord{
		EpisodeID:   episodeID,
		Scenario:    duelScenarioName,
		Seed:        episodeSeed,
		StartedAt:   startedAt,
		EndedAt:     time.Now().UTC(),
		TicksTotal:  ticksExecuted,
		Outcome:     outcome,
		TotalReward: totalReward,
	})
}

func normalizedDuelRunConfig(config DuelRunConfig) DuelRunConfig {
	if config.Episodes <= 0 {
		config.Episodes = 10
	}
	if config.MaxTicksPerEpisode <= 0 {
		config.MaxTicksPerEpisode = 600
	}
	if config.WorldColumns <= 0 {
		config.WorldColumns = 64
	}
	if config.WorldRows <= 0 {
		config.WorldRows = 64
	}
	if config.TileSize <= 0 {
		config.TileSize = 16
	}
	return config
}

func duelLayout(rng *rand.Rand, gameWorld world.World) (geom.Point, geom.Point, []geom.Point) {
	centerTileX := gameWorld.Columns() / 2
	centerTileY := gameWorld.Rows() / 2
	targetOffsetY := rng.Intn(7) - 3
	shooterTileX := centerTileX - 10
	shooterTileY := centerTileY
	targetTileX := centerTileX + 8
	targetTileY := centerTileY + targetOffsetY

	topWaypoint := cellAnchor(targetTileX, targetTileY-5, gameWorld.TileSize())
	bottomWaypoint := cellAnchor(targetTileX, targetTileY+5, gameWorld.TileSize())
	return cellAnchor(shooterTileX, shooterTileY, gameWorld.TileSize()),
		cellAnchor(targetTileX, targetTileY, gameWorld.TileSize()),
		[]geom.Point{topWaypoint, bottomWaypoint}
}

func decideShooterAction(snapshot unit.DuelSnapshot, previousTargetPos geom.Point, hasPreviousTargetPos bool) (string, geom.Point) {
	if !snapshot.Shooter.Alive || !snapshot.Target.Alive {
		return "none", geom.Point{}
	}
	if !snapshot.Shooter.WeaponReady || snapshot.Shooter.HasActiveFireOrder || snapshot.Shooter.HasQueuedFireOrder {
		return "none", geom.Point{}
	}
	if snapshot.DistanceToTarget <= 1e-6 {
		return "none", geom.Point{}
	}

	direction := snapshot.RelativeTarget
	if hasPreviousTargetPos {
		targetVelocity := geom.Point{
			X: snapshot.Target.Position.X - previousTargetPos.X,
			Y: snapshot.Target.Position.Y - previousTargetPos.Y,
		}
		direction.X += targetVelocity.X * 2
		direction.Y += targetVelocity.Y * 2
	}

	length := math.Hypot(direction.X, direction.Y)
	if length <= 1e-6 {
		return "none", geom.Point{}
	}

	return "fire", geom.Point{
		X: direction.X / length,
		Y: direction.Y / length,
	}
}

func targetMoveReportFinished(reports []unit.OrderReport) bool {
	for _, report := range reports {
		if report.Kind != unit.OrderKindMove {
			continue
		}
		switch report.Status {
		case unit.OrderCompleted, unit.OrderFailed, unit.OrderCanceled:
			return true
		}
	}
	return false
}

func resolvePostTickSnapshot(manager *unit.Manager, shooterID, targetID int64, fallback unit.DuelSnapshot, combatEvents []unit.CombatEvent) unit.DuelSnapshot {
	if manager != nil {
		if snapshot, ok := manager.DuelSnapshot(shooterID, targetID); ok {
			return snapshot
		}
	}

	resolved := fallback
	if manager != nil {
		if shooterSnapshot, ok := manager.UnitSnapshot(shooterID); ok {
			resolved.Shooter = shooterSnapshot
		}
		if targetSnapshot, ok := manager.UnitSnapshot(targetID); ok {
			resolved.Target = targetSnapshot
		}
	}

	for _, event := range combatEvents {
		if event.Type != unit.CombatEventUnitKilled {
			continue
		}
		switch event.TargetUnitID {
		case shooterID:
			resolved.Shooter.Alive = false
			resolved.Shooter.Health = 0
		case targetID:
			resolved.Target.Alive = false
			resolved.Target.Health = 0
		}
	}

	resolved.RelativeTarget = geom.Point{
		X: resolved.Target.Position.X - resolved.Shooter.Position.X,
		Y: resolved.Target.Position.Y - resolved.Shooter.Position.Y,
	}
	resolved.DistanceToTarget = math.Hypot(resolved.RelativeTarget.X, resolved.RelativeTarget.Y)
	return resolved
}

func rewardForTick(shooterID, targetID int64, events []unit.CombatEvent) float32 {
	reward := float32(-0.001)
	for _, event := range events {
		switch event.Type {
		case unit.CombatEventProjectileHit:
			if event.SourceUnitID == shooterID {
				reward += 1
			}
			if event.TargetUnitID == shooterID {
				reward -= 1
			}
		case unit.CombatEventProjectileExpired:
			if event.SourceUnitID == shooterID {
				reward -= 0.05
			}
		case unit.CombatEventUnitKilled:
			if event.SourceUnitID == shooterID && event.TargetUnitID == targetID {
				reward += 5
			}
			if event.TargetUnitID == shooterID {
				reward -= 5
			}
		}
	}
	return reward
}

func orderReportsToEventRecords(episodeID uint64, tick int64, reports []unit.OrderReport, createdAt time.Time) []EventRecord {
	if len(reports) == 0 {
		return nil
	}

	records := make([]EventRecord, 0, len(reports))
	for _, report := range reports {
		records = append(records, EventRecord{
			EpisodeID:   episodeID,
			Tick:        uint32(tick),
			Category:    "order",
			EventType:   "order_" + report.Status.String(),
			UnitID:      report.UnitID,
			OrderID:     report.OrderID,
			OrderKind:   report.Kind.String(),
			OrderStatus: report.Status.String(),
			PositionX:   float32(report.TargetPoint.X),
			PositionY:   float32(report.TargetPoint.Y),
			DirectionX:  float32(report.Direction.X),
			DirectionY:  float32(report.Direction.Y),
			CreatedAt:   createdAt,
		})
	}
	return records
}

func combatEventsToEventRecords(episodeID uint64, events []unit.CombatEvent, createdAt time.Time) []EventRecord {
	if len(events) == 0 {
		return nil
	}

	records := make([]EventRecord, 0, len(events))
	for _, event := range events {
		records = append(records, EventRecord{
			EpisodeID:        episodeID,
			Tick:             uint32(maxInt64(event.Tick, 0)),
			Category:         "combat",
			EventType:        string(event.Type),
			SourceUnitID:     event.SourceUnitID,
			TargetUnitID:     event.TargetUnitID,
			ProjectileUnitID: event.ProjectileUnitID,
			PositionX:        float32(event.Position.X),
			PositionY:        float32(event.Position.Y),
			Damage:           int16(event.Damage),
			Killed:           boolToUInt8(event.Killed),
			CreatedAt:        createdAt,
		})
	}
	return records
}

func cellAnchor(tileX, tileY int, tileSize float64) geom.Point {
	return geom.Point{
		X: (float64(tileX) + 0.5) * tileSize,
		Y: (float64(tileY) + 0.5) * tileSize,
	}
}

func boolToUInt8(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}

func maxInt(value, min int) int {
	if value < min {
		return min
	}
	return value
}

func maxInt64(value, min int64) int64 {
	if value < min {
		return min
	}
	return value
}
