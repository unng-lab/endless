package rl

import (
	"context"
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
		"CREATE VIEW default.endless_rl_transitions AS",
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

func TestLoadClickHouseConfigFromEnvUsesBuiltInDefaultsWhenEnvIsEmpty(t *testing.T) {
	cfg := LoadClickHouseConfigFromEnv(func(string) string { return "" })

	if len(cfg.Addr) != 1 || cfg.Addr[0] != defaultClickHouseAddr {
		t.Fatalf("LoadClickHouseConfigFromEnv() Addr = %#v, want [%q]", cfg.Addr, defaultClickHouseAddr)
	}
	if cfg.Database != defaultClickHouseDB {
		t.Fatalf("LoadClickHouseConfigFromEnv() Database = %q, want %q", cfg.Database, defaultClickHouseDB)
	}
	if cfg.Username != defaultClickHouseUser {
		t.Fatalf("LoadClickHouseConfigFromEnv() Username = %q, want %q", cfg.Username, defaultClickHouseUser)
	}
	if cfg.Password != defaultClickHousePass {
		t.Fatalf("LoadClickHouseConfigFromEnv() Password = %q, want %q", cfg.Password, defaultClickHousePass)
	}
}

func TestClickHouseRecorderTranslatesLocalEpisodeIDsBeforeBuffering(t *testing.T) {
	recorder := &ClickHouseRecorder{
		cfg: ClickHouseConfig{
			BatchSize: 128,
		},
		episodeIDBase: 200,
		steps:         make([]StepRecord, 0, 8),
		events:        make([]EventRecord, 0, 8),
		episodes:      make([]EpisodeRecord, 0, 8),
	}

	if err := recorder.RecordStep(context.TODO(), StepRecord{EpisodeID: 1, Tick: 7}); err != nil {
		t.Fatalf("RecordStep() error = %v", err)
	}
	if err := recorder.RecordEvents(context.TODO(), []EventRecord{{EpisodeID: 1, Tick: 7}}); err != nil {
		t.Fatalf("RecordEvents() error = %v", err)
	}
	if err := recorder.RecordEpisode(context.TODO(), EpisodeRecord{EpisodeID: 1}); err != nil {
		t.Fatalf("RecordEpisode() error = %v", err)
	}

	if got, want := recorder.steps[0].EpisodeID, uint64(201); got != want {
		t.Fatalf("buffered step EpisodeID = %d, want %d", got, want)
	}
	if got, want := recorder.events[0].EpisodeID, uint64(201); got != want {
		t.Fatalf("buffered event EpisodeID = %d, want %d", got, want)
	}
	if got, want := recorder.episodes[0].EpisodeID, uint64(201); got != want {
		t.Fatalf("buffered episode EpisodeID = %d, want %d", got, want)
	}
}
