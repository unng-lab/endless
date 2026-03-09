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
	}
}

func (u *Unit) Update(delta float64) {
	u.elapsed += delta
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
