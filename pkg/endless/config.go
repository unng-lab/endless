package endless

import gamescenario "github.com/unng-lab/endless/pkg/endless/scenario"

// GameConfig groups constructor options for Game so callers may choose the startup scenario
// without exposing the scenario package from every call site explicitly.
type GameConfig struct {
	Mode gamescenario.Mode
}

// normalizedGameConfig applies stable defaults once so every launcher path builds the game
// through the same constructor and only overrides the scenario mode when necessary.
func normalizedGameConfig(config GameConfig) GameConfig {
	if config.Mode == "" {
		config.Mode = gamescenario.ModeBasic
	}

	return config
}
