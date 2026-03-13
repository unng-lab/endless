package gomlxtrain

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/unng-lab/endless/pkg/rl"
)

func TestPrepareDatasetBuildsConcatenatedCriticInputsFromJSONL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "transitions.jsonl")

	recordOne := sampleTrainerRecord(1, 1, 1, 0)
	recordTwo := sampleTrainerRecord(1, 2, 2, 1)
	writeTrainingJSONLFile(t, path, recordOne, recordTwo)

	dataset, err := PrepareDataset(context.Background(), Config{
		Source: InputSourceConfig{
			Format: InputFormatJSONL,
			Path:   path,
		},
		BatchSize:    2,
		Epochs:       1,
		LearningRate: 0.001,
		Discount:     0.5,
		HiddenDims:   []int{8},
	})
	if err != nil {
		t.Fatalf("PrepareDataset() error = %v", err)
	}

	if got, want := dataset.Samples, 2; got != want {
		t.Fatalf("dataset.Samples = %d, want %d", got, want)
	}
	if got, want := len(dataset.Inputs), dataset.Samples*dataset.InputDim; got != want {
		t.Fatalf("len(dataset.Inputs) = %d, want %d", got, want)
	}
	if got, want := len(dataset.Targets), dataset.Samples; got != want {
		t.Fatalf("len(dataset.Targets) = %d, want %d", got, want)
	}
	if got, want := dataset.Targets[0], float32(2); got != want {
		t.Fatalf("dataset.Targets[0] = %f, want %f", got, want)
	}
	if got, want := dataset.Targets[1], float32(2); got != want {
		t.Fatalf("dataset.Targets[1] = %f, want %f", got, want)
	}
	if got, want := dataset.TerminalTransitions, 1; got != want {
		t.Fatalf("dataset.TerminalTransitions = %d, want %d", got, want)
	}
	if got, want := dataset.ContinuedTransitions, 1; got != want {
		t.Fatalf("dataset.ContinuedTransitions = %d, want %d", got, want)
	}
}

func writeTrainingJSONLFile(t *testing.T, path string, records ...rl.TrainingTransitionRecord) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	encoder := json.NewEncoder(file)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			t.Fatalf("Encode(record) error = %v", err)
		}
	}
}

func sampleTrainerRecord(episodeID uint64, tick uint32, reward float32, done uint8) rl.TrainingTransitionRecord {
	record := sampleTrainingTransitionRecordForTrainer()
	record.EpisodeID = episodeID
	record.Tick = tick
	record.Reward = reward
	record.Done = done
	return record
}

func sampleTrainingTransitionRecordForTrainer() rl.TrainingTransitionRecord {
	terrainPatch := make([]int16, 25)
	occupancyPatch := make([]int16, 25)
	for index := range terrainPatch {
		terrainPatch[index] = 0
		occupancyPatch[index] = 0
	}
	return rl.TrainingTransitionRecord{
		EpisodeID: 1,
		Tick:      1,
		Scenario:  rl.DuelScenarioWithCover,
		Outcome:   "timeout",

		ObsPatchRadius:                  2,
		ObsShooterX:                     256,
		ObsShooterY:                     128,
		ObsShooterHP:                    3,
		ObsTargetX:                      512,
		ObsTargetY:                      256,
		ObsTargetHP:                     2,
		ObsRelativeTargetX:              128,
		ObsRelativeTargetY:              64,
		ObsDistanceToTarget:             143,
		ObsProjectileCount:              1,
		ObsShooterWeaponReady:           1,
		ObsShooterCooldownRemaining:     0,
		ObsShooterHasActiveFireOrder:    0,
		ObsShooterHasQueuedFireOrder:    0,
		ObsShooterHasActiveMoveOrder:    0,
		ObsShooterHasQueuedMoveOrder:    0,
		ObsShooterHasDestination:        0,
		ObsShooterDestinationX:          0,
		ObsShooterDestinationY:          0,
		ObsShooterDistanceToDestination: 0,
		ObsShooterRecentMoveFailure:     0,
		ObsLocalTerrainPatch:            terrainPatch,
		ObsLocalOccupancyPatch:          occupancyPatch,
		ObsNearestFriendlyShotExists:    0,
		ObsNearestFriendlyShotX:         0,
		ObsNearestFriendlyShotY:         0,
		ObsNearestFriendlyShotDist:      0,
		ObsNearestHostileShotExists:     0,
		ObsNearestHostileShotX:          0,
		ObsNearestHostileShotY:          0,
		ObsNearestHostileShotDist:       0,

		ActionType:        string(rl.ActionTypeMove),
		ActionAccepted:    1,
		ActionMoveTargetX: 320,
		ActionMoveTargetY: 160,
		ActionDirX:        0,
		ActionDirY:        0,
		Reward:            1,
		Done:              0,

		NextObsPatchRadius:                  2,
		NextObsShooterX:                     272,
		NextObsShooterY:                     128,
		NextObsShooterHP:                    3,
		NextObsTargetX:                      512,
		NextObsTargetY:                      256,
		NextObsTargetHP:                     2,
		NextObsRelativeTargetX:              112,
		NextObsRelativeTargetY:              64,
		NextObsDistanceToTarget:             129,
		NextObsProjectileCount:              1,
		NextObsShooterWeaponReady:           1,
		NextObsShooterCooldownRemaining:     0,
		NextObsShooterHasActiveFireOrder:    0,
		NextObsShooterHasQueuedFireOrder:    0,
		NextObsShooterHasActiveMoveOrder:    1,
		NextObsShooterHasQueuedMoveOrder:    0,
		NextObsShooterHasDestination:        1,
		NextObsShooterDestinationX:          48,
		NextObsShooterDestinationY:          32,
		NextObsShooterDistanceToDestination: 57,
		NextObsShooterRecentMoveFailure:     0,
		NextObsLocalTerrainPatch:            terrainPatch,
		NextObsLocalOccupancyPatch:          occupancyPatch,
		NextObsNearestFriendlyShotExists:    0,
		NextObsNearestFriendlyShotX:         0,
		NextObsNearestFriendlyShotY:         0,
		NextObsNearestFriendlyShotDist:      0,
		NextObsNearestHostileShotExists:     0,
		NextObsNearestHostileShotX:          0,
		NextObsNearestHostileShotY:          0,
		NextObsNearestHostileShotDist:       0,
	}
}
