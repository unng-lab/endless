package scenario

import (
	"github.com/unng-lab/endless/pkg/rl"
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
	ModeRLDuel Mode = "rl_duel"
)

// Scenario captures the small contract required by the game loop to seed the initial world
// state, advance any scenario-specific orchestration each tick and optionally expose one debug
// line for the overlay.
type Scenario interface {
	SeedUnits(manager *unit.Manager)
	Update(gameTick int64, manager *unit.Manager)
	DebugText() string
}

// Config groups every scenario-side option the game constructor may pass into the selected
// bootstrapper without exposing individual scenario internals to the launcher layer.
type Config struct {
	Mode   Mode
	RLDuel rl.VisualDuelScenarioConfig
}

// New chooses the concrete scene bootstrapper for the requested launch mode. Falling back to
// the basic scene keeps accidental zero-value configs lightweight and safe.
func New(config Config, gameWorld world.World) (Scenario, error) {
	switch config.Mode {
	case ModeStress:
		return newStressScenario(gameWorld), nil
	case ModeRLDuel:
		return rl.NewVisualDuelScenario(gameWorld, config.RLDuel)
	case ModeBasic:
		fallthrough
	default:
		return newBasicScenario(gameWorld), nil
	}
}
