package unit

import (
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

type Kind string

const (
	KindRunner        Kind = "runner"
	KindRunnerFocused Kind = "runnerfocused"
)

var runnerAnimation = Animation{
	FrameCount:    8,
	FrameDuration: 0.1,
}

type Unit struct {
	Position geom.Point
	Kind     Kind

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

func (u *Unit) Update(delta float64) {
	u.elapsed += delta
	u.advance(delta)
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
	u.path = append(u.path[:0], path...)
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

func (u *Unit) advance(delta float64) {
	if delta <= 0 || len(u.path) == 0 || u.moveSpeed <= 0 {
		return
	}

	remaining := u.moveSpeed * delta
	for remaining > 0 && len(u.path) > 0 {
		target := u.path[0]
		dx := target.X - u.Position.X
		dy := target.Y - u.Position.Y
		distance := math.Hypot(dx, dy)
		if distance <= 1e-6 {
			u.Position = target
			u.path = u.path[1:]
			continue
		}
		if distance <= remaining {
			u.Position = target
			u.path = u.path[1:]
			remaining -= distance
			continue
		}

		factor := remaining / distance
		u.Position.X += dx * factor
		u.Position.Y += dy * factor
		return
	}
}
