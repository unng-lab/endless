package unit

import (
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

type Kind string

const (
	KindRunner        Kind = "runner"
	KindRunnerFocused Kind = "runnerfocused"
	KindWall          Kind = "wall"
	KindBarricade     Kind = "barricade"
)

var runnerAnimation = Animation{
	FrameCount:    8,
	FrameDuration: 0.1,
}

type Unit struct {
	Position geom.Point
	Kind     Kind
	OnScreen bool

	animation Animation
	elapsed   float64
	moveSpeed float64
	path      []geom.Point
}

func NewRunner(position geom.Point, focused bool, phase float64) Unit {
	kind := KindRunner
	if focused {
		kind = KindRunnerFocused
	}

	return Unit{
		Position:  position,
		Kind:      kind,
		animation: runnerAnimation,
		elapsed:   phase,
		moveSpeed: 48,
	}
}

func NewWall(position geom.Point) Unit {
	return Unit{
		Position:  position,
		Kind:      KindWall,
		animation: Animation{FrameCount: 1, FrameDuration: 1},
	}
}

func NewBarricade(position geom.Point) Unit {
	return Unit{
		Position:  position,
		Kind:      KindBarricade,
		animation: Animation{FrameCount: 1, FrameDuration: 1},
	}
}

func (u *Unit) Update(delta float64, speedMultiplier func(geom.Point) float64) {
	u.elapsed += delta
	u.advance(delta, speedMultiplier)
}

func (u Unit) Frame() int {
	return u.animation.frameAt(u.elapsed)
}

func (u Unit) Name() string {
	switch u.Kind {
	case KindRunner:
		return "Runner"
	case KindRunnerFocused:
		return "Runner Focused"
	case KindWall:
		return "Wall"
	case KindBarricade:
		return "Barricade"
	default:
		return string(u.Kind)
	}
}

func (u Unit) TilePosition(tileSize float64) (int, int) {
	if tileSize <= 0 {
		return 0, 0
	}

	return int(math.Floor(u.Position.X / tileSize)), int(math.Floor(u.Position.Y / tileSize))
}

func (u *Unit) SetPath(path []geom.Point) {
	if !u.IsMobile() {
		u.path = u.path[:0]
		return
	}

	u.path = append(u.path[:0], path...)
}

func (u Unit) IsMobile() bool {
	return u.moveSpeed > 0
}

func (u Unit) BlocksMovement() bool {
	switch u.Kind {
	case KindWall, KindBarricade:
		return true
	default:
		return false
	}
}

func (u Unit) HasPath() bool {
	return len(u.path) > 0
}

func (u Unit) PathLen() int {
	return len(u.path)
}

func (u Unit) Destination() (geom.Point, bool) {
	if len(u.path) == 0 {
		return geom.Point{}, false
	}

	return u.path[len(u.path)-1], true
}

func (u *Unit) advance(delta float64, speedMultiplier func(geom.Point) float64) {
	if delta <= 0 || len(u.path) == 0 || u.moveSpeed <= 0 {
		return
	}

	remainingTime := delta
	for remainingTime > 0 && len(u.path) > 0 {
		currentSpeed := u.moveSpeed
		if speedMultiplier != nil {
			multiplier := speedMultiplier(u.Position)
			if multiplier <= 0 {
				return
			}
			currentSpeed *= multiplier
		}
		if currentSpeed <= 0 {
			return
		}

		target := u.path[0]
		dx := target.X - u.Position.X
		dy := target.Y - u.Position.Y
		distance := math.Hypot(dx, dy)
		if distance <= 1e-6 {
			u.Position = target
			u.path = u.path[1:]
			continue
		}

		timeToTarget := distance / currentSpeed
		if timeToTarget <= remainingTime {
			u.Position = target
			u.path = u.path[1:]
			remainingTime -= timeToTarget
			continue
		}

		distanceToTravel := currentSpeed * remainingTime
		factor := distanceToTravel / distance
		u.Position.X += dx * factor
		u.Position.Y += dy * factor
		return
	}
}
