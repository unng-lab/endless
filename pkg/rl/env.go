package rl

import (
	"time"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/unit"
)

// ActionType names the small control surface exposed to RL actors. The environment intentionally
// maps these actions back into the existing manager order API so training code never bypasses
// gameplay rules for pathfinding, wind-up or cooldown handling.
type ActionType string

const (
	ActionTypeNone ActionType = "none"
	ActionTypeMove ActionType = "move"
	ActionTypeFire ActionType = "fire"
)

// Action carries one requested gameplay intent. Move actions use MoveTarget, fire actions use
// FireDirection, and no-op actions keep both payloads empty.
type Action struct {
	Type          ActionType
	MoveTarget    geom.Point
	FireDirection geom.Point
}

// Observation keeps the policy-facing state for one decision point. The embedded duel snapshot
// remains the canonical world projection, while the extra metadata provides enough context for
// movement target generation without forcing policies to reach back into world internals.
type Observation struct {
	Snapshot             unit.DuelSnapshot
	PreviousTargetPos    geom.Point
	HasPreviousTargetPos bool
	TileSize             float64
	WorldWidth           float64
	WorldHeight          float64
}

// StepResult groups the post-tick state transition emitted by one environment step so dataset
// writers may persist both dense step rows and sparse event rows from the same source of truth.
type StepResult struct {
	After          Observation
	Reward         float32
	Done           bool
	Outcome        string
	ShooterReports []unit.OrderReport
	TargetReports  []unit.OrderReport
	CombatEvents   []unit.CombatEvent
	CreatedAt      time.Time
}

// Environment describes the minimal lifecycle needed by headless RL collection code.
// Observations are read before an action is applied, then Step advances the simulation by one
// gameplay tick and returns the resulting transition.
type Environment interface {
	Reset(seed int64) (Observation, error)
	Observe() (Observation, error)
	ApplyAction(Action) (bool, error)
	Step() (StepResult, error)
	Close()
}

// Policy picks one gameplay action from the current observation. Separating it from the
// environment keeps scripted baselines, random actors and later learned policies interchangeable.
type Policy interface {
	ChooseAction(Observation) Action
}
