package rl

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/unng-lab/endless/pkg/geom"
)

const (
	PolicyLeadAndStrafe = "lead_strafe"
	PolicyRandom        = "random"
)

// NewPolicyByName centralizes policy construction so collection and evaluation mode use the
// same actor registry and the same deterministic seed handling for stochastic policies.
func NewPolicyByName(name string, seed int64) (Policy, error) {
	switch normalizedPolicyName(name) {
	case "", PolicyLeadAndStrafe:
		policy := NewLeadAndStrafePolicy()
		return policy, nil
	case PolicyRandom:
		return NewRandomPolicy(seed), nil
	default:
		return nil, fmt.Errorf("unsupported policy %q", name)
	}
}

func normalizedPolicyName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// RandomPolicy issues a small mix of random move and fire commands that still respect the same
// gameplay API surface. It is mainly intended for bootstrap datasets and evaluation baselines.
type RandomPolicy struct {
	rng *rand.Rand
}

// NewRandomPolicy builds one deterministic random actor from the provided seed.
func NewRandomPolicy(seed int64) *RandomPolicy {
	return &RandomPolicy{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// ChooseAction samples one legal-looking action from the current duel observation. The policy
// avoids spamming move orders while one is already active and only fires when the weapon is ready.
func (p *RandomPolicy) ChooseAction(observation Observation) Action {
	if p == nil || p.rng == nil {
		return Action{Type: ActionTypeNone}
	}

	snapshot := observation.Snapshot
	if !snapshot.Shooter.Alive || !snapshot.Target.Alive {
		return Action{Type: ActionTypeNone}
	}

	if snapshot.Shooter.WeaponReady && !snapshot.Shooter.HasActiveFireOrder && !snapshot.Shooter.HasQueuedFireOrder && p.rng.Float64() < 0.35 {
		direction := noisyDirectionTowardsTarget(snapshot.RelativeTarget, p.rng)
		if direction != (geom.Point{}) {
			return Action{
				Type:          ActionTypeFire,
				FireDirection: direction,
			}
		}
	}

	if snapshot.Shooter.HasActiveMoveOrder || snapshot.Shooter.HasQueuedMoveOrder {
		return Action{Type: ActionTypeNone}
	}
	if p.rng.Float64() < 0.45 {
		return Action{
			Type:       ActionTypeMove,
			MoveTarget: randomMoveTarget(observation, p.rng),
		}
	}

	return Action{Type: ActionTypeNone}
}

func noisyDirectionTowardsTarget(relativeTarget geom.Point, rng *rand.Rand) geom.Point {
	if rng == nil {
		return geom.Point{}
	}

	direction := geom.Point{
		X: relativeTarget.X + rng.Float64()*20 - 10,
		Y: relativeTarget.Y + rng.Float64()*20 - 10,
	}
	length := math.Hypot(direction.X, direction.Y)
	if length <= 1e-6 {
		return geom.Point{}
	}

	return geom.Point{
		X: direction.X / length,
		Y: direction.Y / length,
	}
}

func randomMoveTarget(observation Observation, rng *rand.Rand) geom.Point {
	if rng == nil {
		return geom.Point{}
	}

	center := observation.Snapshot.Target.Position
	radius := observation.TileSize * (4 + rng.Float64()*6)
	angle := rng.Float64() * math.Pi * 2
	target := geom.Point{
		X: center.X + math.Cos(angle)*radius,
		Y: center.Y + math.Sin(angle)*radius,
	}
	return clampPointToWorld(target, observation.WorldWidth, observation.WorldHeight)
}
