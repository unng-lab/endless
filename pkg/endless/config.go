package endless

import (
	gamescenario "github.com/unng-lab/endless/pkg/endless/scenario"
	"github.com/unng-lab/endless/pkg/rl"
	"github.com/unng-lab/endless/pkg/world"
)

const (
	defaultWorldColumns  = 10000
	defaultWorldRows     = 10000
	defaultWorldTileSize = 16.0

	rlDuelWorldColumns = 256
	rlDuelWorldRows    = 256
)

// GameConfig groups constructor options for Game so callers may choose the startup scenario
// without exposing the scenario package from every call site explicitly.
type GameConfig struct {
	Mode   gamescenario.Mode
	RLDuel rl.VisualDuelScenarioConfig
}

// normalizedGameConfig applies stable defaults once so every launcher path builds the game
// through the same constructor and only overrides the scenario mode when necessary.
func normalizedGameConfig(config GameConfig) GameConfig {
	if config.Mode == "" {
		config.Mode = gamescenario.ModeBasic
	}

	return config
}

// worldConfig resolves the exact world dimensions that should back one launcher mode. The
// visual RL duel intentionally mirrors the headless training environment dimensions so runtime
// policy features stay inside the same coordinate ranges used during offline training.
func (config GameConfig) worldConfig() world.Config {
	switch config.Mode {
	case gamescenario.ModeRLDuel:
		return world.Config{
			Columns:  rlDuelWorldColumns,
			Rows:     rlDuelWorldRows,
			TileSize: defaultWorldTileSize,
		}
	default:
		return world.Config{
			Columns:  defaultWorldColumns,
			Rows:     defaultWorldRows,
			TileSize: defaultWorldTileSize,
		}
	}
}
