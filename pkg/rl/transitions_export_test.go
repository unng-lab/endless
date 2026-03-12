package rl

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildTrainingTransitionsSelectQueryUsesExplicitTrainerFacingContract(t *testing.T) {
	query, args := buildTrainingTransitionsSelectQuery(
		ClickHouseConfig{
			Database:    "default",
			TablePrefix: "endless_rl",
		},
		TrainingTransitionQuery{
			Scenario:     "duel_with_cover",
			Outcome:      "target_killed",
			EpisodeIDMin: 10,
			EpisodeIDMax: 20,
			Limit:        128,
		},
	)

	expectedFragments := []string{
		"SELECT",
		"obs_patch_radius",
		"next_obs_local_terrain_patch",
		"created_at",
		"FROM default.endless_rl_transitions",
		"scenario = $1",
		"outcome = $2",
		"episode_id >= $3",
		"episode_id <= $4",
		"ORDER BY episode_id ASC, tick ASC",
		"LIMIT $5",
	}
	for _, fragment := range expectedFragments {
		if strings.Contains(query, fragment) {
			continue
		}
		t.Fatalf("buildTrainingTransitionsSelectQuery() missing fragment %q", fragment)
	}

	expectedArgs := []any{"duel_with_cover", "target_killed", uint64(10), uint64(20), 128}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Fatalf("buildTrainingTransitionsSelectQuery() args = %#v, want %#v", args, expectedArgs)
	}
}

func TestMustTrainingTransitionSelectColumnsCoversDocumentedContract(t *testing.T) {
	if len(trainingTransitionSelectColumns) != reflect.TypeOf(TrainingTransitionRecord{}).NumField() {
		t.Fatalf(
			"trainingTransitionSelectColumns length = %d, want %d",
			len(trainingTransitionSelectColumns),
			reflect.TypeOf(TrainingTransitionRecord{}).NumField(),
		)
	}

	expectedColumns := []string{
		"episode_id",
		"obs_local_occupancy_patch",
		"action_type",
		"next_obs_nearest_hostile_shot_dist",
		"created_at",
	}
	for _, column := range expectedColumns {
		found := false
		for _, actual := range trainingTransitionSelectColumns {
			if actual != column {
				continue
			}
			found = true
			break
		}
		if found {
			continue
		}
		t.Fatalf("trainingTransitionSelectColumns missing column %q", column)
	}
}

func TestNormalizedTrainingTransitionQuerySwapsInvertedEpisodeBounds(t *testing.T) {
	normalized := normalizedTrainingTransitionQuery(TrainingTransitionQuery{
		EpisodeIDMin: 99,
		EpisodeIDMax: 5,
		Limit:        -1,
	})

	if normalized.EpisodeIDMin != 5 || normalized.EpisodeIDMax != 99 {
		t.Fatalf("normalizedTrainingTransitionQuery() bounds = [%d,%d], want [5,99]", normalized.EpisodeIDMin, normalized.EpisodeIDMax)
	}
	if normalized.Limit != 0 {
		t.Fatalf("normalizedTrainingTransitionQuery() limit = %d, want 0", normalized.Limit)
	}
}
