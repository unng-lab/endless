package rl

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/unit"
)

// DuelRunConfig describes one headless data-generation session.
type DuelRunConfig struct {
	Episodes           int
	MaxTicksPerEpisode int64
	Seed               int64
	WorldColumns       int
	WorldRows          int
	TileSize           float64
	Scenario           string
}

// RunDuelCollection executes deterministic duel episodes and streams their resulting
// observation/action/reward traces into the supplied recorder.
func RunDuelCollection(ctx context.Context, config DuelRunConfig, recorder Recorder) error {
	return RunDuelCollectionWithPolicy(ctx, config, recorder, NewLeadAndStrafePolicy())
}

// RunDuelCollectionWithPolicy executes deterministic duel episodes with the supplied shooter
// policy and streams the resulting observation/action/reward traces into the supplied recorder.
func RunDuelCollectionWithPolicy(ctx context.Context, config DuelRunConfig, recorder Recorder, policy Policy) error {
	config = normalizedDuelRunConfig(config)
	if recorder == nil {
		recorder = NoopRecorder{}
	}
	if policy == nil {
		policy = NewLeadAndStrafePolicy()
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
		if err := runOneDuelEpisode(ctx, episodeID, episodeSeed, config, recorder, policy); err != nil {
			return fmt.Errorf("run duel episode %d: %w", episodeID, err)
		}
	}

	return recorder.Close(ctx)
}

func runOneDuelEpisode(ctx context.Context, episodeID uint64, episodeSeed int64, config DuelRunConfig, recorder Recorder, policy Policy) error {
	environment := NewDuelEnvironment(config)
	defer environment.Close()

	startedAt := time.Now().UTC()
	totalReward := float32(0)
	outcome := "setup_failed"
	ticksExecuted := uint32(0)

	before, err := environment.Reset(episodeSeed)
	if err != nil {
		return err
	}

	for tick := int64(1); tick <= config.MaxTicksPerEpisode; tick++ {
		action := policy.ChooseAction(before)
		actionAccepted, actionErr := environment.ApplyAction(action)
		if actionErr != nil {
			actionAccepted = false
		}

		stepResult, err := environment.Step()
		if err != nil {
			return err
		}
		ticksExecuted = uint32(stepResult.After.Snapshot.Tick)
		outcome = stepResult.Outcome
		totalReward += stepResult.Reward

		eventRecords := append(
			orderReportsToEventRecords(episodeID, tick, stepResult.ShooterReports, stepResult.CreatedAt),
			orderReportsToEventRecords(episodeID, tick, stepResult.TargetReports, stepResult.CreatedAt)...,
		)
		eventRecords = append(eventRecords, combatEventsToEventRecords(episodeID, stepResult.CombatEvents, stepResult.CreatedAt)...)
		if err := recorder.RecordEvents(ctx, eventRecords); err != nil {
			return err
		}

		stepRecord := buildStepRecord(episodeID, before, stepResult.After, action, actionAccepted, stepResult.Reward, stepResult.Done, stepResult.CreatedAt)
		if err := recorder.RecordStep(ctx, stepRecord); err != nil {
			return err
		}

		before = stepResult.After
		if stepResult.Done {
			break
		}
	}

	return recorder.RecordEpisode(ctx, EpisodeRecord{
		EpisodeID:   episodeID,
		Scenario:    config.Scenario,
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
	config.Scenario = normalizedDuelScenarioName(config.Scenario)
	return config
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

func buildStepRecord(episodeID uint64, before, after Observation, action Action, actionAccepted bool, reward float32, done bool, createdAt time.Time) StepRecord {
	return StepRecord{
		EpisodeID:                       episodeID,
		Tick:                            uint32(after.Snapshot.Tick),
		ShooterID:                       after.Snapshot.Shooter.UnitID,
		TargetID:                        after.Snapshot.Target.UnitID,
		ObsPatchRadius:                  int16(before.PatchRadius),
		ObsShooterX:                     float32(before.Snapshot.Shooter.Position.X),
		ObsShooterY:                     float32(before.Snapshot.Shooter.Position.Y),
		ObsShooterHP:                    int16(before.Snapshot.Shooter.Health),
		ObsTargetX:                      float32(before.Snapshot.Target.Position.X),
		ObsTargetY:                      float32(before.Snapshot.Target.Position.Y),
		ObsTargetHP:                     int16(before.Snapshot.Target.Health),
		ObsRelativeTargetX:              float32(before.Snapshot.RelativeTarget.X),
		ObsRelativeTargetY:              float32(before.Snapshot.RelativeTarget.Y),
		ObsDistanceToTarget:             float32(before.Snapshot.DistanceToTarget),
		ObsProjectileCount:              uint16(before.Snapshot.ProjectileCount),
		ObsShooterWeaponReady:           boolToUInt8(before.Snapshot.Shooter.WeaponReady),
		ObsShooterCooldownRemaining:     uint16(maxInt(before.Snapshot.Shooter.FireCooldownRemaining, 0)),
		ObsShooterHasActiveFireOrder:    boolToUInt8(before.Snapshot.Shooter.HasActiveFireOrder),
		ObsShooterHasQueuedFireOrder:    boolToUInt8(before.Snapshot.Shooter.HasQueuedFireOrder),
		ObsShooterHasActiveMoveOrder:    boolToUInt8(before.Snapshot.Shooter.HasActiveMoveOrder),
		ObsShooterHasQueuedMoveOrder:    boolToUInt8(before.Snapshot.Shooter.HasQueuedMoveOrder),
		ObsShooterHasDestination:        boolToUInt8(before.ShooterHasDestination),
		ObsShooterDestinationX:          float32(before.ShooterDestinationRelativeX),
		ObsShooterDestinationY:          float32(before.ShooterDestinationRelativeY),
		ObsShooterDistanceToDestination: float32(before.ShooterDistanceToDestination),
		ObsShooterRecentMoveFailure:     boolToUInt8(before.ShooterRecentMoveFailure),
		ObsLocalTerrainPatch:            append([]int16(nil), before.LocalTerrainPatch...),
		ObsLocalOccupancyPatch:          append([]int16(nil), before.LocalOccupancyPatch...),
		ObsNearestFriendlyShotExists:    boolToUInt8(before.NearestFriendlyShot.Exists),
		ObsNearestFriendlyShotX:         float32(before.NearestFriendlyShot.RelativeX),
		ObsNearestFriendlyShotY:         float32(before.NearestFriendlyShot.RelativeY),
		ObsNearestFriendlyShotDist:      float32(before.NearestFriendlyShot.Distance),
		ObsNearestHostileShotExists:     boolToUInt8(before.NearestHostileShot.Exists),
		ObsNearestHostileShotX:          float32(before.NearestHostileShot.RelativeX),
		ObsNearestHostileShotY:          float32(before.NearestHostileShot.RelativeY),
		ObsNearestHostileShotDist:       float32(before.NearestHostileShot.Distance),
		PatchRadius:                     int16(after.PatchRadius),
		ShooterX:                        float32(after.Snapshot.Shooter.Position.X),
		ShooterY:                        float32(after.Snapshot.Shooter.Position.Y),
		ShooterHP:                       int16(after.Snapshot.Shooter.Health),
		TargetX:                         float32(after.Snapshot.Target.Position.X),
		TargetY:                         float32(after.Snapshot.Target.Position.Y),
		TargetHP:                        int16(after.Snapshot.Target.Health),
		RelativeTargetX:                 float32(after.Snapshot.RelativeTarget.X),
		RelativeTargetY:                 float32(after.Snapshot.RelativeTarget.Y),
		DistanceToTarget:                float32(after.Snapshot.DistanceToTarget),
		ProjectileCount:                 uint16(after.Snapshot.ProjectileCount),
		ShooterWeaponReady:              boolToUInt8(after.Snapshot.Shooter.WeaponReady),
		ShooterCooldownRemaining:        uint16(maxInt(after.Snapshot.Shooter.FireCooldownRemaining, 0)),
		ShooterHasActiveFireOrder:       boolToUInt8(after.Snapshot.Shooter.HasActiveFireOrder),
		ShooterHasQueuedFireOrder:       boolToUInt8(after.Snapshot.Shooter.HasQueuedFireOrder),
		ShooterHasActiveMoveOrder:       boolToUInt8(after.Snapshot.Shooter.HasActiveMoveOrder),
		ShooterHasQueuedMoveOrder:       boolToUInt8(after.Snapshot.Shooter.HasQueuedMoveOrder),
		ShooterHasDestination:           boolToUInt8(after.ShooterHasDestination),
		ShooterDestinationX:             float32(after.ShooterDestinationRelativeX),
		ShooterDestinationY:             float32(after.ShooterDestinationRelativeY),
		ShooterDistanceToDestination:    float32(after.ShooterDistanceToDestination),
		ShooterRecentMoveFailure:        boolToUInt8(after.ShooterRecentMoveFailure),
		LocalTerrainPatch:               append([]int16(nil), after.LocalTerrainPatch...),
		LocalOccupancyPatch:             append([]int16(nil), after.LocalOccupancyPatch...),
		NearestFriendlyShotExists:       boolToUInt8(after.NearestFriendlyShot.Exists),
		NearestFriendlyShotX:            float32(after.NearestFriendlyShot.RelativeX),
		NearestFriendlyShotY:            float32(after.NearestFriendlyShot.RelativeY),
		NearestFriendlyShotDist:         float32(after.NearestFriendlyShot.Distance),
		NearestHostileShotExists:        boolToUInt8(after.NearestHostileShot.Exists),
		NearestHostileShotX:             float32(after.NearestHostileShot.RelativeX),
		NearestHostileShotY:             float32(after.NearestHostileShot.RelativeY),
		NearestHostileShotDist:          float32(after.NearestHostileShot.Distance),
		ActionType:                      string(action.Type),
		ActionAccepted:                  boolToUInt8(actionAccepted),
		ActionMoveTargetX:               float32(action.MoveTarget.X),
		ActionMoveTargetY:               float32(action.MoveTarget.Y),
		ActionDirX:                      float32(action.FireDirection.X),
		ActionDirY:                      float32(action.FireDirection.Y),
		Reward:                          reward,
		Done:                            boolToUInt8(done),
		CreatedAt:                       createdAt,
	}
}
