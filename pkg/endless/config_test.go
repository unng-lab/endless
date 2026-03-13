package endless

import (
	"testing"

	gamescenario "github.com/unng-lab/endless/pkg/endless/scenario"
)

// TestGameConfigWorldConfigKeepsVisualRLDuelOnTrainingSizedWorld verifies that the rendered
// RL duel uses the same world dimensions as the headless trainer environment so runtime model
// features stay inside the coordinate ranges seen during offline training.
func TestGameConfigWorldConfigKeepsVisualRLDuelOnTrainingSizedWorld(t *testing.T) {
	tests := []struct {
		name        string
		config      GameConfig
		wantColumns int
		wantRows    int
	}{
		{
			name:        "basic keeps large desktop world",
			config:      GameConfig{Mode: gamescenario.ModeBasic},
			wantColumns: defaultWorldColumns,
			wantRows:    defaultWorldRows,
		},
		{
			name:        "rl duel switches to training sized world",
			config:      GameConfig{Mode: gamescenario.ModeRLDuel},
			wantColumns: rlDuelWorldColumns,
			wantRows:    rlDuelWorldRows,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			worldConfig := tc.config.worldConfig()
			if worldConfig.Columns != tc.wantColumns {
				t.Fatalf("worldConfig().Columns = %d, want %d", worldConfig.Columns, tc.wantColumns)
			}
			if worldConfig.Rows != tc.wantRows {
				t.Fatalf("worldConfig().Rows = %d, want %d", worldConfig.Rows, tc.wantRows)
			}
			if worldConfig.TileSize != defaultWorldTileSize {
				t.Fatalf("worldConfig().TileSize = %.1f, want %.1f", worldConfig.TileSize, defaultWorldTileSize)
			}
		})
	}
}
