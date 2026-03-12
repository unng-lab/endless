package rl

import (
	"strings"
	"testing"
)

func TestClickHouseRecorderTransitionsViewStatementExposesTrainerAliases(t *testing.T) {
	recorder := &ClickHouseRecorder{
		cfg: ClickHouseConfig{
			Database:    "default",
			TablePrefix: "endless_rl",
		},
	}

	statement := recorder.transitionsViewStatement()
	expectedFragments := []string{
		"CREATE VIEW IF NOT EXISTS default.endless_rl_transitions AS",
		"LEFT JOIN default.endless_rl_episodes AS e ON s.episode_id = e.episode_id",
		"s.action_type",
		"s.patch_radius AS next_obs_patch_radius",
		"s.shooter_x AS next_obs_shooter_x",
		"s.shooter_has_destination AS next_obs_shooter_has_destination",
		"s.shooter_recent_move_failure AS next_obs_shooter_recent_move_failure",
		"s.local_terrain_patch AS next_obs_local_terrain_patch",
		"s.nearest_hostile_shot_dist AS next_obs_nearest_hostile_shot_dist",
	}
	for _, fragment := range expectedFragments {
		if strings.Contains(statement, fragment) {
			continue
		}
		t.Fatalf("transitionsViewStatement() missing fragment %q", fragment)
	}
}
