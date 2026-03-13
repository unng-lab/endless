package rl

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

const defaultVisualDuelMaxTicks = 1200

// VisualDuelScenarioConfig groups every launcher-facing knob needed to render one RL-driven
// duel inside the regular Ebiten game window.
type VisualDuelScenarioConfig struct {
	Scenario  string
	Policy    string
	Seed      int64
	ModelPath string
	MaxTicks  int64
}

// VisualDuelScenario drives one rendered duel with the same observation and action surface as
// the headless RL environment so developers may watch policy behaviour in the desktop client.
type VisualDuelScenario struct {
	world  world.World
	config VisualDuelScenarioConfig
	policy Policy

	shooterID            int64
	targetID             int64
	targetWaypoints      []geom.Point
	nextTargetWaypoint   int
	targetMoveInFlight   bool
	previousTargetPos    geom.Point
	hasPreviousTargetPos bool
	recentMoveFailure    bool
	lastAction           Action
	lastActionAccepted   bool
	lastOutcome          string
	lastTick             int64
	lastObservation      Observation
	hasLastObservation   bool
	lastPolicyDebug      string
	done                 bool
	spawnedUnits         int
	staticObjects        int
	runtimePolicyLabel   string
}

// NewVisualDuelScenario validates the requested policy source and prepares one visual duel
// controller that plugs into the existing scene interface used by the desktop launcher.
func NewVisualDuelScenario(gameWorld world.World, config VisualDuelScenarioConfig) (*VisualDuelScenario, error) {
	config = normalizedVisualDuelScenarioConfig(config)
	policy, policyLabel, err := newVisualDuelPolicy(config)
	if err != nil {
		return nil, err
	}

	return &VisualDuelScenario{
		world:              gameWorld,
		config:             config,
		policy:             policy,
		lastAction:         Action{Type: ActionTypeNone},
		lastActionAccepted: true,
		lastOutcome:        "in_progress",
		runtimePolicyLabel: policyLabel,
	}, nil
}

func normalizedVisualDuelScenarioConfig(config VisualDuelScenarioConfig) VisualDuelScenarioConfig {
	config.Scenario = normalizedDuelScenarioName(config.Scenario)
	if config.Policy == "" {
		config.Policy = PolicyLeadAndStrafe
	}
	if config.MaxTicks <= 0 {
		config.MaxTicks = defaultVisualDuelMaxTicks
	}
	return config
}

func newVisualDuelPolicy(config VisualDuelScenarioConfig) (Policy, string, error) {
	if config.ModelPath != "" {
		policy, policyLabel, err := LoadRuntimePolicyFromPath(config.ModelPath)
		if err != nil {
			return nil, "", fmt.Errorf("load runtime policy: %w", err)
		}
		return policy, policyLabel, nil
	}

	policy, err := NewPolicyByName(config.Policy, config.Seed)
	if err != nil {
		return nil, "", err
	}
	return policy, normalizedPolicyName(config.Policy), nil
}

// SeedUnits creates the shooter, the scripted target and any static cover needed by the chosen
// duel layout, then locks the unit selection onto the shooter for easier visual inspection.
func (s *VisualDuelScenario) SeedUnits(manager *unit.Manager) {
	if s == nil || manager == nil {
		return
	}

	layout := buildDuelScenarioLayout(rand.New(rand.NewSource(s.config.Seed)), s.config.Scenario, s.world)
	for _, staticUnit := range layout.StaticUnits {
		manager.AddUnit(staticUnit)
		s.staticObjects++
	}

	s.shooterID = manager.AddUnit(unit.NewRunner(layout.ShooterSpawn, false, 0))
	s.targetID = manager.AddUnit(unit.NewRunner(layout.TargetSpawn, true, 6))
	s.targetWaypoints = append([]geom.Point(nil), layout.TargetWaypoints...)
	s.spawnedUnits = 2
	s.lastOutcome = "in_progress"
	s.lastAction = Action{Type: ActionTypeNone}
	s.lastActionAccepted = true
	s.lastObservation = Observation{}
	s.hasLastObservation = false
	s.lastPolicyDebug = ""
	s.done = false
	manager.SelectUnitByID(s.shooterID)
}

// Update advances one rendered duel tick: it consumes reports from the previous simulation
// step, rebuilds the policy observation, issues the next action and maintains target patrol.
func (s *VisualDuelScenario) Update(gameTick int64, manager *unit.Manager) {
	if s == nil || manager == nil || s.done {
		return
	}

	s.lastTick = gameTick
	manager.SelectUnitByID(s.shooterID)

	shooterReports := manager.DrainUnitOrderReports(s.shooterID)
	targetReports := manager.DrainUnitOrderReports(s.targetID)
	combatEvents := manager.DrainCombatEvents()
	if targetMoveReportFinished(targetReports) {
		s.targetMoveInFlight = false
	}
	s.recentMoveFailure = shooterMoveFailed(shooterReports)

	observation, err := s.observe(manager)
	if err != nil {
		s.lastOutcome = "observation_failed"
		s.done = true
		return
	}
	s.lastObservation = observation
	s.hasLastObservation = true

	s.lastOutcome = visualDuelOutcome(observation, gameTick, s.config.MaxTicks)
	if s.lastOutcome != "in_progress" {
		s.done = true
		s.previousTargetPos = observation.Snapshot.Target.Position
		s.hasPreviousTargetPos = observation.Snapshot.Target.Alive
		_ = combatEvents
		return
	}

	action := s.policy.ChooseAction(observation)
	s.lastPolicyDebug = policyDebugText(s.policy)
	actionAccepted := s.applyAction(manager, action)
	s.lastAction = action
	s.lastActionAccepted = actionAccepted
	s.issueTargetPatrolOrder(manager)

	s.previousTargetPos = observation.Snapshot.Target.Position
	s.hasPreviousTargetPos = observation.Snapshot.Target.Alive
}

// DebugText exposes the current duel state and the last chosen action in the on-screen overlay.
func (s *VisualDuelScenario) DebugText() string {
	if s == nil {
		return ""
	}

	observationText := "obs: unavailable"
	if s.hasLastObservation {
		snapshot := s.lastObservation.Snapshot
		desiredRange := NewLeadAndStrafePolicy().desiredRange() * s.lastObservation.TileSize
		observationText = fmt.Sprintf(
			"obs: dist %.1f desired %.1f shooter_hp %d target_hp %d weapon_ready %t move_active %t fire_active %t",
			snapshot.DistanceToTarget,
			desiredRange,
			snapshot.Shooter.Health,
			snapshot.Target.Health,
			snapshot.Shooter.WeaponReady,
			snapshot.Shooter.HasActiveMoveOrder || snapshot.Shooter.HasQueuedMoveOrder,
			snapshot.Shooter.HasActiveFireOrder || snapshot.Shooter.HasQueuedFireOrder,
		)
	}

	return fmt.Sprintf(
		"Scene: rl_duel  policy %s  layout %s  outcome %s  tick %d/%d  shooter %d  target %d\n%s\nlast_action: %s accepted %t\n%s",
		s.runtimePolicyLabel,
		s.config.Scenario,
		s.lastOutcome,
		s.lastTick,
		s.config.MaxTicks,
		s.shooterID,
		s.targetID,
		observationText,
		formatVisualDuelAction(s.lastAction),
		s.lastActionAccepted,
		composeVisualPolicyDebugText(s.lastPolicyDebug, s.staticObjects),
	)
}

func (s *VisualDuelScenario) observe(manager *unit.Manager) (Observation, error) {
	if s == nil || manager == nil {
		return Observation{}, fmt.Errorf("visual duel scenario is not initialized")
	}

	snapshot, ok := manager.DuelSnapshot(s.shooterID, s.targetID)
	if !ok {
		return Observation{}, fmt.Errorf("visual duel snapshot is unavailable")
	}

	return buildObservation(
		s.world,
		snapshot,
		manager.ProjectileSnapshots(),
		manager.BlockingUnitSnapshots(),
		s.previousTargetPos,
		s.hasPreviousTargetPos,
		s.recentMoveFailure,
	), nil
}

func (s *VisualDuelScenario) applyAction(manager *unit.Manager, action Action) bool {
	if s == nil || manager == nil {
		return false
	}

	switch action.Type {
	case "", ActionTypeNone:
		return true
	case ActionTypeMove:
		return manager.IssueMoveOrder(s.shooterID, action.MoveTarget) == nil
	case ActionTypeFire:
		return manager.IssueFireOrder(s.shooterID, action.FireDirection) == nil
	default:
		return false
	}
}

func (s *VisualDuelScenario) issueTargetPatrolOrder(manager *unit.Manager) {
	if s == nil || manager == nil || s.targetMoveInFlight || len(s.targetWaypoints) == 0 {
		return
	}

	targetPoint := s.targetWaypoints[s.nextTargetWaypoint]
	if err := manager.IssueMoveOrder(s.targetID, targetPoint); err != nil {
		return
	}

	s.targetMoveInFlight = true
	s.nextTargetWaypoint = (s.nextTargetWaypoint + 1) % len(s.targetWaypoints)
}

func visualDuelOutcome(observation Observation, tick, maxTicks int64) string {
	switch {
	case !observation.Snapshot.Target.Alive:
		return "target_killed"
	case !observation.Snapshot.Shooter.Alive:
		return "shooter_killed"
	case maxTicks > 0 && tick >= maxTicks:
		return "timeout"
	default:
		return "in_progress"
	}
}

type runtimeDecisionDebugProvider interface {
	LastDecisionDebugText() string
}

func policyDebugText(policy Policy) string {
	if provider, ok := policy.(runtimeDecisionDebugProvider); ok {
		return provider.LastDecisionDebugText()
	}
	return ""
}

func composeVisualPolicyDebugText(policyDebug string, staticObjects int) string {
	if policyDebug == "" {
		return fmt.Sprintf("policy_debug: unavailable  static %d", staticObjects)
	}
	return fmt.Sprintf("%s  static %d", policyDebug, staticObjects)
}

func formatVisualDuelAction(action Action) string {
	switch action.Type {
	case ActionTypeMove:
		return fmt.Sprintf("move target=(%.1f, %.1f)", action.MoveTarget.X, action.MoveTarget.Y)
	case ActionTypeFire:
		angle := math.Atan2(action.FireDirection.Y, action.FireDirection.X) * 180 / math.Pi
		return fmt.Sprintf("fire dir=(%.2f, %.2f) angle=%.0fdeg", action.FireDirection.X, action.FireDirection.Y, angle)
	case ActionTypeNone, "":
		fallthrough
	default:
		return "none"
	}
}
