package rl

import (
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

const (
	defaultDesiredRangeTiles = 9.0
	defaultStrafeOffsetTiles = 3.0
	defaultFireRangeTiles    = 12.0
)

// LeadAndStrafePolicy is the scripted baseline used for dataset generation before any learned
// policy exists. It aims shots with simple target leading and issues movement orders that keep
// the shooter near one preferred engagement radius.
type LeadAndStrafePolicy struct {
	DesiredRangeTiles float64
	StrafeOffsetTiles float64
	FireRangeTiles    float64
}

// NewLeadAndStrafePolicy builds one baseline actor with conservative defaults that fit the
// projectile range and the current duel layout.
func NewLeadAndStrafePolicy() LeadAndStrafePolicy {
	return LeadAndStrafePolicy{
		DesiredRangeTiles: defaultDesiredRangeTiles,
		StrafeOffsetTiles: defaultStrafeOffsetTiles,
		FireRangeTiles:    defaultFireRangeTiles,
	}
}

// ChooseAction emits either a fire order when the target is in range and the weapon is ready
// or one move order that adjusts the shooter back towards the preferred firing ring.
func (p LeadAndStrafePolicy) ChooseAction(observation Observation) Action {
	snapshot := observation.Snapshot
	if !snapshot.Shooter.Alive || !snapshot.Target.Alive {
		return Action{Type: ActionTypeNone}
	}

	if fireAction, ok := p.chooseFireAction(observation); ok {
		return fireAction
	}

	if snapshot.Shooter.HasActiveMoveOrder || snapshot.Shooter.HasQueuedMoveOrder {
		return Action{Type: ActionTypeNone}
	}

	moveTarget, ok := p.chooseMoveTarget(observation)
	if !ok {
		return Action{Type: ActionTypeNone}
	}

	return Action{
		Type:       ActionTypeMove,
		MoveTarget: moveTarget,
	}
}

func (p LeadAndStrafePolicy) chooseFireAction(observation Observation) (Action, bool) {
	snapshot := observation.Snapshot
	if !snapshot.Shooter.WeaponReady || snapshot.Shooter.HasActiveFireOrder || snapshot.Shooter.HasQueuedFireOrder {
		return Action{}, false
	}
	if snapshot.DistanceToTarget <= 1e-6 {
		return Action{}, false
	}
	if observation.TileSize > 0 && snapshot.DistanceToTarget > p.fireRange()*observation.TileSize {
		return Action{}, false
	}

	direction := snapshot.RelativeTarget
	if observation.HasPreviousTargetPos {
		targetVelocity := geom.Point{
			X: snapshot.Target.Position.X - observation.PreviousTargetPos.X,
			Y: snapshot.Target.Position.Y - observation.PreviousTargetPos.Y,
		}
		direction.X += targetVelocity.X * 2
		direction.Y += targetVelocity.Y * 2
	}

	length := math.Hypot(direction.X, direction.Y)
	if length <= 1e-6 {
		return Action{}, false
	}

	return Action{
		Type: ActionTypeFire,
		FireDirection: geom.Point{
			X: direction.X / length,
			Y: direction.Y / length,
		},
	}, true
}

func (p LeadAndStrafePolicy) chooseMoveTarget(observation Observation) (geom.Point, bool) {
	snapshot := observation.Snapshot
	distance := snapshot.DistanceToTarget
	if distance <= 1e-6 {
		return geom.Point{}, false
	}

	radial := geom.Point{
		X: (snapshot.Shooter.Position.X - snapshot.Target.Position.X) / distance,
		Y: (snapshot.Shooter.Position.Y - snapshot.Target.Position.Y) / distance,
	}
	perpendicular := geom.Point{X: -radial.Y, Y: radial.X}
	desiredRange := p.desiredRange() * observation.TileSize
	strafeOffset := p.strafeOffset() * observation.TileSize

	baseTarget := geom.Point{
		X: snapshot.Target.Position.X + radial.X*desiredRange,
		Y: snapshot.Target.Position.Y + radial.Y*desiredRange,
	}

	// When the shooter is already near the preferred radius, alternate between two mirrored
	// lateral offsets so the baseline still exercises move-order collection while weaving.
	moveTarget := baseTarget
	if observation.TileSize > 0 && math.Abs(distance-desiredRange) <= observation.TileSize*1.5 {
		strafeSign := 1.0
		if (snapshot.Tick/45)%2 == 1 {
			strafeSign = -1
		}
		moveTarget.X += perpendicular.X * strafeOffset * strafeSign
		moveTarget.Y += perpendicular.Y * strafeOffset * strafeSign
	}

	return clampPointToWorld(moveTarget, observation.WorldWidth, observation.WorldHeight), true
}

func (p LeadAndStrafePolicy) desiredRange() float64 {
	if p.DesiredRangeTiles > 0 {
		return p.DesiredRangeTiles
	}
	return defaultDesiredRangeTiles
}

func (p LeadAndStrafePolicy) strafeOffset() float64 {
	if p.StrafeOffsetTiles > 0 {
		return p.StrafeOffsetTiles
	}
	return defaultStrafeOffsetTiles
}

func (p LeadAndStrafePolicy) fireRange() float64 {
	if p.FireRangeTiles > 0 {
		return p.FireRangeTiles
	}
	return defaultFireRangeTiles
}

func clampPointToWorld(point geom.Point, worldWidth, worldHeight float64) geom.Point {
	clamped := point
	if worldWidth > 0 {
		clamped.X = geom.ClampFloat(clamped.X, 0, worldWidth-1e-6)
	}
	if worldHeight > 0 {
		clamped.Y = geom.ClampFloat(clamped.Y, 0, worldHeight-1e-6)
	}
	return clamped
}
