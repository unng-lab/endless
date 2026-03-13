package rl

import (
	"fmt"
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

// LinearQStubRuntimePolicy scores a compact set of legal-looking action candidates with one
// trained linear model so developers may visualize the current smoke-check learner in-game.
type LinearQStubRuntimePolicy struct {
	artifact LinearQStubArtifact
	fallback Policy

	lastDecisionDebug string
}

// LoadLinearQStubRuntimePolicy restores one saved stub model from disk and prepares it for
// action scoring against live duel observations.
func LoadLinearQStubRuntimePolicy(path string) (*LinearQStubRuntimePolicy, error) {
	artifact, err := LoadLinearQStubArtifact(path)
	if err != nil {
		return nil, err
	}

	return NewLinearQStubRuntimePolicy(artifact)
}

// NewLinearQStubRuntimePolicy validates the supplied artifact and keeps one scripted fallback
// so runtime gameplay stays functional even if a later scoring step encounters bad inputs.
func NewLinearQStubRuntimePolicy(artifact LinearQStubArtifact) (*LinearQStubRuntimePolicy, error) {
	if err := artifact.Validate(); err != nil {
		return nil, err
	}

	return &LinearQStubRuntimePolicy{
		artifact: artifact,
		fallback: NewLeadAndStrafePolicy(),
	}, nil
}

// ChooseAction scores a small deterministic action set and returns the best-valued candidate.
// The candidate generator deliberately stays conservative so the smoke-check runtime policy
// only emits actions that already resemble the training distribution.
func (p *LinearQStubRuntimePolicy) ChooseAction(observation Observation) Action {
	if p == nil {
		return Action{Type: ActionTypeNone}
	}

	spec := p.artifact.NormalizationSpec.Normalized()
	obsVector, err := vectorizeRuntimeObservation(observation, spec)
	if err != nil {
		p.lastDecisionDebug = "model: observation vectorization failed, fallback policy used"
		return p.fallbackAction(observation)
	}

	candidates := buildLinearQStubActionCandidates(observation)
	if len(candidates) == 0 {
		p.lastDecisionDebug = "model: no candidates, fallback policy used"
		return p.fallbackAction(observation)
	}

	bestAction := candidates[0]
	bestScore := float32(math.Inf(-1))
	successfulCandidates := 0
	for _, candidate := range candidates {
		actionVector, err := vectorizeRuntimeAction(spec, candidate, true)
		if err != nil {
			continue
		}

		score, err := p.artifact.Model.Predict(obsVector, actionVector)
		if err != nil {
			continue
		}
		successfulCandidates++
		if score > bestScore {
			bestScore = score
			bestAction = candidate
		}
	}

	if math.IsInf(float64(bestScore), -1) {
		p.lastDecisionDebug = "model: candidate scoring failed, fallback policy used"
		return p.fallbackAction(observation)
	}
	p.lastDecisionDebug = fmt.Sprintf(
		"model: best_score=%.4f candidates=%d scored=%d chosen=%s",
		bestScore,
		len(candidates),
		successfulCandidates,
		actionKey(bestAction),
	)
	return bestAction
}

// LastDecisionDebugText exposes one compact summary of the last runtime scoring pass so the
// visual duel overlay may show why the current model chose its most recent action.
func (p *LinearQStubRuntimePolicy) LastDecisionDebugText() string {
	if p == nil {
		return ""
	}
	return p.lastDecisionDebug
}

func (p *LinearQStubRuntimePolicy) fallbackAction(observation Observation) Action {
	if p == nil || p.fallback == nil {
		return Action{Type: ActionTypeNone}
	}
	return p.fallback.ChooseAction(observation)
}

func vectorizeRuntimeObservation(observation Observation, spec TransitionNormalizationSpec) ([]float32, error) {
	spec = spec.Normalized()
	return vectorizeObservationProjection(spec, transitionObservationProjection{
		PatchRadius:                  int16(observation.PatchRadius),
		ShooterX:                     float32(observation.Snapshot.Shooter.Position.X),
		ShooterY:                     float32(observation.Snapshot.Shooter.Position.Y),
		ShooterHP:                    int16(observation.Snapshot.Shooter.Health),
		TargetX:                      float32(observation.Snapshot.Target.Position.X),
		TargetY:                      float32(observation.Snapshot.Target.Position.Y),
		TargetHP:                     int16(observation.Snapshot.Target.Health),
		RelativeTargetX:              float32(observation.Snapshot.RelativeTarget.X),
		RelativeTargetY:              float32(observation.Snapshot.RelativeTarget.Y),
		DistanceToTarget:             float32(observation.Snapshot.DistanceToTarget),
		ProjectileCount:              uint16(maxInt(observation.Snapshot.ProjectileCount, 0)),
		ShooterWeaponReady:           boolToUInt8(observation.Snapshot.Shooter.WeaponReady),
		ShooterCooldownRemaining:     uint16(maxInt(observation.Snapshot.Shooter.FireCooldownRemaining, 0)),
		ShooterHasActiveFireOrder:    boolToUInt8(observation.Snapshot.Shooter.HasActiveFireOrder),
		ShooterHasQueuedFireOrder:    boolToUInt8(observation.Snapshot.Shooter.HasQueuedFireOrder),
		ShooterHasActiveMoveOrder:    boolToUInt8(observation.Snapshot.Shooter.HasActiveMoveOrder),
		ShooterHasQueuedMoveOrder:    boolToUInt8(observation.Snapshot.Shooter.HasQueuedMoveOrder),
		ShooterHasDestination:        boolToUInt8(observation.ShooterHasDestination),
		ShooterDestinationX:          float32(observation.ShooterDestinationRelativeX),
		ShooterDestinationY:          float32(observation.ShooterDestinationRelativeY),
		ShooterDistanceToDestination: float32(observation.ShooterDistanceToDestination),
		ShooterRecentMoveFailure:     boolToUInt8(observation.ShooterRecentMoveFailure),
		LocalTerrainPatch:            append([]int16(nil), observation.LocalTerrainPatch...),
		LocalOccupancyPatch:          append([]int16(nil), observation.LocalOccupancyPatch...),
		NearestFriendlyShotExists:    boolToUInt8(observation.NearestFriendlyShot.Exists),
		NearestFriendlyShotX:         float32(observation.NearestFriendlyShot.RelativeX),
		NearestFriendlyShotY:         float32(observation.NearestFriendlyShot.RelativeY),
		NearestFriendlyShotDist:      float32(observation.NearestFriendlyShot.Distance),
		NearestHostileShotExists:     boolToUInt8(observation.NearestHostileShot.Exists),
		NearestHostileShotX:          float32(observation.NearestHostileShot.RelativeX),
		NearestHostileShotY:          float32(observation.NearestHostileShot.RelativeY),
		NearestHostileShotDist:       float32(observation.NearestHostileShot.Distance),
	})
}

func vectorizeRuntimeAction(spec TransitionNormalizationSpec, action Action, accepted bool) ([]float32, error) {
	spec = spec.Normalized()
	record := TrainingTransitionRecord{
		ActionType:        string(action.Type),
		ActionAccepted:    boolToUInt8(accepted),
		ActionMoveTargetX: float32(action.MoveTarget.X),
		ActionMoveTargetY: float32(action.MoveTarget.Y),
		ActionDirX:        float32(action.FireDirection.X),
		ActionDirY:        float32(action.FireDirection.Y),
	}
	return vectorizeAction(spec, record)
}

func buildLinearQStubActionCandidates(observation Observation) []Action {
	candidates := make([]Action, 0, 5)
	if !observation.Snapshot.Shooter.Alive || !observation.Snapshot.Target.Alive {
		return []Action{{Type: ActionTypeNone}}
	}

	baseline := NewLeadAndStrafePolicy()
	moveCandidates := buildLinearQStubMoveCandidates(observation, baseline)
	forceMove := shouldForceRuntimeMove(observation, baseline, moveCandidates)
	if !forceMove {
		candidates = append(candidates, Action{Type: ActionTypeNone})
	}
	if fireAction, ok := baseline.chooseFireAction(observation); ok {
		candidates = append(candidates, fireAction)
	}
	candidates = append(candidates, moveCandidates...)
	if len(candidates) == 0 {
		return []Action{{Type: ActionTypeNone}}
	}
	return dedupePolicyActions(candidates)
}

func buildLinearQStubMoveCandidates(observation Observation, baseline LeadAndStrafePolicy) []Action {
	snapshot := observation.Snapshot
	if snapshot.Shooter.HasActiveMoveOrder || snapshot.Shooter.HasQueuedMoveOrder {
		return nil
	}
	if !snapshot.Shooter.Alive || !snapshot.Target.Alive {
		return nil
	}

	distance := snapshot.DistanceToTarget
	if distance <= 1e-6 || observation.TileSize <= 0 {
		return nil
	}

	radial := geom.Point{
		X: (snapshot.Shooter.Position.X - snapshot.Target.Position.X) / distance,
		Y: (snapshot.Shooter.Position.Y - snapshot.Target.Position.Y) / distance,
	}
	perpendicular := geom.Point{X: -radial.Y, Y: radial.X}
	desiredRange := baseline.desiredRange() * observation.TileSize
	strafeOffset := baseline.strafeOffset() * observation.TileSize

	baseTarget := clampPointToWorld(geom.Point{
		X: snapshot.Target.Position.X + radial.X*desiredRange,
		Y: snapshot.Target.Position.Y + radial.Y*desiredRange,
	}, observation.WorldWidth, observation.WorldHeight)
	positiveTarget := clampPointToWorld(geom.Point{
		X: baseTarget.X + perpendicular.X*strafeOffset,
		Y: baseTarget.Y + perpendicular.Y*strafeOffset,
	}, observation.WorldWidth, observation.WorldHeight)
	negativeTarget := clampPointToWorld(geom.Point{
		X: baseTarget.X - perpendicular.X*strafeOffset,
		Y: baseTarget.Y - perpendicular.Y*strafeOffset,
	}, observation.WorldWidth, observation.WorldHeight)

	candidates := []Action{
		{Type: ActionTypeMove, MoveTarget: baseTarget},
		{Type: ActionTypeMove, MoveTarget: positiveTarget},
		{Type: ActionTypeMove, MoveTarget: negativeTarget},
	}
	if preferred, ok := baseline.chooseMoveTarget(observation); ok {
		candidates = append([]Action{{Type: ActionTypeMove, MoveTarget: preferred}}, candidates...)
	}
	return candidates
}

func shouldForceRuntimeMove(observation Observation, baseline LeadAndStrafePolicy, moveCandidates []Action) bool {
	if len(moveCandidates) == 0 {
		return false
	}

	snapshot := observation.Snapshot
	if snapshot.Shooter.HasActiveMoveOrder || snapshot.Shooter.HasQueuedMoveOrder {
		return false
	}
	if !snapshot.Shooter.Alive || !snapshot.Target.Alive {
		return false
	}

	canFireNow := false
	if _, ok := baseline.chooseFireAction(observation); ok {
		canFireNow = true
	}
	if canFireNow {
		return false
	}

	desiredRange := baseline.desiredRange() * observation.TileSize
	if desiredRange <= 0 {
		return true
	}
	return math.Abs(snapshot.DistanceToTarget-desiredRange) > observation.TileSize
}

func dedupePolicyActions(actions []Action) []Action {
	if len(actions) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(actions))
	deduped := make([]Action, 0, len(actions))
	for _, action := range actions {
		key := actionKey(action)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, action)
	}
	return deduped
}

func actionKey(action Action) string {
	return fmt.Sprintf(
		"%s|%.3f|%.3f|%.3f|%.3f",
		action.Type,
		action.MoveTarget.X,
		action.MoveTarget.Y,
		action.FireDirection.X,
		action.FireDirection.Y,
	)
}
