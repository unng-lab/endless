package endless

import (
	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

// GameMode selects which scene bootstrap logic should populate the world before the Ebiten
// loop starts. The default launcher uses the lightweight setup, while the dedicated stress cmd
// opts into the heavy profiling harness explicitly.
type GameMode string

const (
	GameModeBasic  GameMode = "basic"
	GameModeStress GameMode = "stress"
)

// GameConfig groups constructor options for Game so callers may choose the startup scenario
// without needing separate package-level globals.
type GameConfig struct {
	Mode GameMode
}

// gameScenario captures the small contract required by Game to seed the initial world state,
// advance any scenario-specific orchestration each tick and optionally expose one debug line.
type gameScenario interface {
	SeedUnits(manager *unit.Manager)
	Update(gameTick int64, manager *unit.Manager)
	DebugText() string
}

// normalizedGameConfig applies stable defaults once so every launcher path builds the game
// through the same constructor and only overrides the scenario mode when necessary.
func normalizedGameConfig(config GameConfig) GameConfig {
	if config.Mode == "" {
		config.Mode = GameModeBasic
	}

	return config
}

// newScenarioForMode chooses the concrete scene bootstrapper for the requested launch mode.
// Falling back to the basic scene keeps accidental zero-value configs lightweight and safe.
func newScenarioForMode(mode GameMode, gameWorld world.World) gameScenario {
	switch mode {
	case GameModeStress:
		return newStressScenario(gameWorld)
	case GameModeBasic:
		fallthrough
	default:
		return newBasicScenario(gameWorld)
	}
}
