package scenario

import (
	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

// Mode selects which scene bootstrap logic should populate the world before the Ebiten loop
// starts. The default launcher uses the lightweight setup, while the dedicated stress cmd opts
// into the heavy profiling harness explicitly.
type Mode string

const (
	ModeBasic  Mode = "basic"
	ModeStress Mode = "stress"
)

// Scenario captures the small contract required by the game loop to seed the initial world
// state, advance any scenario-specific orchestration each tick and optionally expose one debug
// line for the overlay.
type Scenario interface {
	SeedUnits(manager *unit.Manager)
	Update(gameTick int64, manager *unit.Manager)
	DebugText() string
}

// New chooses the concrete scene bootstrapper for the requested launch mode. Falling back to
// the basic scene keeps accidental zero-value configs lightweight and safe.
func New(mode Mode, gameWorld world.World) Scenario {
	switch mode {
	case ModeStress:
		return newStressScenario(gameWorld)
	case ModeBasic:
		fallthrough
	default:
		return newBasicScenario(gameWorld)
	}
}
