package rl

import (
	"context"
	"time"
)

// EpisodeRecord keeps the final metadata of one fully simulated duel episode.
type EpisodeRecord struct {
	EpisodeID   uint64
	Scenario    string
	Seed        int64
	StartedAt   time.Time
	EndedAt     time.Time
	TicksTotal  uint32
	Outcome     string
	TotalReward float32
}

// StepRecord stores one RL transition row with both the pre-action observation and the
// resulting post-tick compact state. This shape keeps ClickHouse rows directly usable for
// offline training without reconstructing state_t from the previous row.
type StepRecord struct {
	EpisodeID                    uint64
	Tick                         uint32
	ShooterID                    int64
	TargetID                     int64
	ObsShooterX                  float32
	ObsShooterY                  float32
	ObsShooterHP                 int16
	ObsTargetX                   float32
	ObsTargetY                   float32
	ObsTargetHP                  int16
	ObsRelativeTargetX           float32
	ObsRelativeTargetY           float32
	ObsDistanceToTarget          float32
	ObsProjectileCount           uint16
	ObsShooterWeaponReady        uint8
	ObsShooterCooldownRemaining  uint16
	ObsShooterHasActiveFireOrder uint8
	ObsShooterHasQueuedFireOrder uint8
	ObsShooterHasActiveMoveOrder uint8
	ObsShooterHasQueuedMoveOrder uint8
	ShooterX                     float32
	ShooterY                     float32
	ShooterHP                    int16
	TargetX                      float32
	TargetY                      float32
	TargetHP                     int16
	RelativeTargetX              float32
	RelativeTargetY              float32
	DistanceToTarget             float32
	ProjectileCount              uint16
	ShooterWeaponReady           uint8
	ShooterCooldownRemaining     uint16
	ShooterHasActiveFireOrder    uint8
	ShooterHasQueuedFireOrder    uint8
	ShooterHasActiveMoveOrder    uint8
	ShooterHasQueuedMoveOrder    uint8
	ActionType                   string
	ActionAccepted               uint8
	ActionMoveTargetX            float32
	ActionMoveTargetY            float32
	ActionDirX                   float32
	ActionDirY                   float32
	Reward                       float32
	Done                         uint8
	CreatedAt                    time.Time
}

// EventRecord keeps sparse order and combat events in one append-only shape so offline tools
// may correlate lifecycle events with step rows without reconstructing them from logs.
type EventRecord struct {
	EpisodeID        uint64
	Tick             uint32
	Category         string
	EventType        string
	UnitID           int64
	SourceUnitID     int64
	TargetUnitID     int64
	ProjectileUnitID int64
	OrderID          int64
	OrderKind        string
	OrderStatus      string
	PositionX        float32
	PositionY        float32
	DirectionX       float32
	DirectionY       float32
	Damage           int16
	Killed           uint8
	CreatedAt        time.Time
}

// Recorder describes the minimal persistence surface required by the headless duel runner.
// Concrete implementations may target ClickHouse, files, or no-op test sinks.
type Recorder interface {
	RecordStep(context.Context, StepRecord) error
	RecordEvents(context.Context, []EventRecord) error
	RecordEpisode(context.Context, EpisodeRecord) error
	Close(context.Context) error
}

// NoopRecorder keeps the runner usable even when callers want to verify simulation logic
// without writing traces anywhere.
type NoopRecorder struct{}

func (NoopRecorder) RecordStep(context.Context, StepRecord) error {
	return nil
}

func (NoopRecorder) RecordEvents(context.Context, []EventRecord) error {
	return nil
}

func (NoopRecorder) RecordEpisode(context.Context, EpisodeRecord) error {
	return nil
}

func (NoopRecorder) Close(context.Context) error {
	return nil
}
