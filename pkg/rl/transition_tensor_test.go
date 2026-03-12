package rl

import "testing"

func TestDefaultTransitionNormalizationSpecDimensionsMatchFeatureNames(t *testing.T) {
	spec := DefaultTransitionNormalizationSpec()

	if got, want := spec.ObservationDim(), 355; got != want {
		t.Fatalf("ObservationDim() = %d, want %d", got, want)
	}
	if got, want := spec.ActionDim(), 8; got != want {
		t.Fatalf("ActionDim() = %d, want %d", got, want)
	}
	if got, want := len(spec.ObservationFeatureNames()), spec.ObservationDim(); got != want {
		t.Fatalf("len(ObservationFeatureNames()) = %d, want %d", got, want)
	}
	if got, want := len(spec.ActionFeatureNames()), spec.ActionDim(); got != want {
		t.Fatalf("len(ActionFeatureNames()) = %d, want %d", got, want)
	}
}

func TestVectorizeTransitionNormalizesScalarsAndOneHotEncodesPatches(t *testing.T) {
	spec := DefaultTransitionNormalizationSpec()
	record := sampleTrainingTransitionRecord()

	transition, err := VectorizeTransition(record, spec)
	if err != nil {
		t.Fatalf("VectorizeTransition() error = %v", err)
	}

	if len(transition.Obs) != spec.ObservationDim() {
		t.Fatalf("len(Obs) = %d, want %d", len(transition.Obs), spec.ObservationDim())
	}
	if len(transition.NextObs) != spec.ObservationDim() {
		t.Fatalf("len(NextObs) = %d, want %d", len(transition.NextObs), spec.ObservationDim())
	}
	if len(transition.Action) != spec.ActionDim() {
		t.Fatalf("len(Action) = %d, want %d", len(transition.Action), spec.ActionDim())
	}

	if transition.Obs[0] != 1 {
		t.Fatalf("patch radius feature = %f, want 1", transition.Obs[0])
	}
	if transition.Obs[1] != 0.5 {
		t.Fatalf("obs shooter x = %f, want 0.5", transition.Obs[1])
	}
	if transition.Obs[7] != -0.25 {
		t.Fatalf("obs relative target x = %f, want -0.25", transition.Obs[7])
	}
	if transition.Obs[12] != 0.5 {
		t.Fatalf("obs cooldown = %f, want 0.5", transition.Obs[12])
	}
	if transition.Obs[22] != 1 {
		t.Fatalf("nearest friendly exists = %f, want 1", transition.Obs[22])
	}

	terrainStart := 30
	if transition.Obs[terrainStart] != 1 {
		t.Fatalf("terrain patch[0] unknown slot = %f, want 1", transition.Obs[terrainStart])
	}
	if transition.Obs[terrainStart+1] != 0 {
		t.Fatalf("terrain patch[0] grass slot = %f, want 0", transition.Obs[terrainStart+1])
	}
	secondTerrainCellStart := terrainStart + len(spec.TerrainVocabulary)
	if transition.Obs[secondTerrainCellStart] != 0 || transition.Obs[secondTerrainCellStart+1] != 1 {
		t.Fatalf("terrain patch[1] encoding = [%f,%f], want [0,1]", transition.Obs[secondTerrainCellStart], transition.Obs[secondTerrainCellStart+1])
	}

	occupancyStart := terrainStart + spec.ExpectedPatchLength()*len(spec.TerrainVocabulary)
	if transition.Obs[occupancyStart] != 1 {
		t.Fatalf("occupancy patch[0] unknown slot = %f, want 1", transition.Obs[occupancyStart])
	}
	secondOccupancyCellStart := occupancyStart + len(spec.OccupancyVocabulary)
	if transition.Obs[secondOccupancyCellStart] != 0 || transition.Obs[secondOccupancyCellStart+1] != 1 {
		t.Fatalf("occupancy patch[1] encoding = [%f,%f], want [0,1]", transition.Obs[secondOccupancyCellStart], transition.Obs[secondOccupancyCellStart+1])
	}

	if transition.Action[2] != 1 {
		t.Fatalf("fire one-hot slot = %f, want 1", transition.Action[2])
	}
	if transition.Action[3] != 1 {
		t.Fatalf("action accepted = %f, want 1", transition.Action[3])
	}
	if transition.Action[4] != 0.25 || transition.Action[5] != 0.5 {
		t.Fatalf("normalized move target = [%f,%f], want [0.25,0.5]", transition.Action[4], transition.Action[5])
	}
	if transition.Action[6] != -0.5 || transition.Action[7] != 0.75 {
		t.Fatalf("normalized fire direction = [%f,%f], want [-0.5,0.75]", transition.Action[6], transition.Action[7])
	}
	if transition.Done != 1 {
		t.Fatalf("done = %f, want 1", transition.Done)
	}
}

func TestTransitionBatchBuilderPacksFixedSizeBatches(t *testing.T) {
	spec := DefaultTransitionNormalizationSpec()
	builder, err := NewTransitionBatchBuilder(spec, 2)
	if err != nil {
		t.Fatalf("NewTransitionBatchBuilder() error = %v", err)
	}

	record := sampleTrainingTransitionRecord()
	if batch, err := builder.AppendRecord(record); err != nil {
		t.Fatalf("AppendRecord(first) error = %v", err)
	} else if batch != nil {
		t.Fatal("AppendRecord(first) returned completed batch, want nil")
	}

	batch, err := builder.AppendRecord(record)
	if err != nil {
		t.Fatalf("AppendRecord(second) error = %v", err)
	}
	if batch == nil {
		t.Fatal("AppendRecord(second) = nil, want completed batch")
	}
	if batch.BatchSize != 2 {
		t.Fatalf("batch.BatchSize = %d, want 2", batch.BatchSize)
	}
	if len(batch.Obs) != 2*spec.ObservationDim() {
		t.Fatalf("len(batch.Obs) = %d, want %d", len(batch.Obs), 2*spec.ObservationDim())
	}
	if len(batch.Action) != 2*spec.ActionDim() {
		t.Fatalf("len(batch.Action) = %d, want %d", len(batch.Action), 2*spec.ActionDim())
	}
	if tail := builder.Flush(); tail != nil {
		t.Fatalf("Flush() = %#v, want nil after full batch emission", tail)
	}
}

func sampleTrainingTransitionRecord() TrainingTransitionRecord {
	terrainPatch := make([]int16, 25)
	occupancyPatch := make([]int16, 25)
	for index := range terrainPatch {
		terrainPatch[index] = 0
		occupancyPatch[index] = 0
	}
	terrainPatch[0] = -1
	occupancyPatch[0] = -1

	nextTerrainPatch := append([]int16(nil), terrainPatch...)
	nextOccupancyPatch := append([]int16(nil), occupancyPatch...)
	nextTerrainPatch[2] = 4
	nextOccupancyPatch[2] = 5

	return TrainingTransitionRecord{
		EpisodeID: 1,
		Tick:      7,
		Scenario:  DuelScenarioWithCover,
		Outcome:   "target_killed",

		ObsPatchRadius:                  2,
		ObsShooterX:                     512,
		ObsShooterY:                     256,
		ObsShooterHP:                    3,
		ObsTargetX:                      640,
		ObsTargetY:                      384,
		ObsTargetHP:                     2,
		ObsRelativeTargetX:              -256,
		ObsRelativeTargetY:              128,
		ObsDistanceToTarget:             320,
		ObsProjectileCount:              4,
		ObsShooterWeaponReady:           1,
		ObsShooterCooldownRemaining:     5,
		ObsShooterHasActiveFireOrder:    0,
		ObsShooterHasQueuedFireOrder:    1,
		ObsShooterHasActiveMoveOrder:    1,
		ObsShooterHasQueuedMoveOrder:    0,
		ObsShooterHasDestination:        1,
		ObsShooterDestinationX:          -128,
		ObsShooterDestinationY:          64,
		ObsShooterDistanceToDestination: 144,
		ObsShooterRecentMoveFailure:     1,
		ObsLocalTerrainPatch:            terrainPatch,
		ObsLocalOccupancyPatch:          occupancyPatch,
		ObsNearestFriendlyShotExists:    1,
		ObsNearestFriendlyShotX:         64,
		ObsNearestFriendlyShotY:         -32,
		ObsNearestFriendlyShotDist:      96,
		ObsNearestHostileShotExists:     1,
		ObsNearestHostileShotX:          -48,
		ObsNearestHostileShotY:          24,
		ObsNearestHostileShotDist:       80,

		ActionType:        string(ActionTypeFire),
		ActionAccepted:    1,
		ActionMoveTargetX: 256,
		ActionMoveTargetY: 512,
		ActionDirX:        -0.5,
		ActionDirY:        0.75,
		Reward:            1.25,
		Done:              1,

		NextObsPatchRadius:                  2,
		NextObsShooterX:                     520,
		NextObsShooterY:                     264,
		NextObsShooterHP:                    3,
		NextObsTargetX:                      630,
		NextObsTargetY:                      390,
		NextObsTargetHP:                     1,
		NextObsRelativeTargetX:              -220,
		NextObsRelativeTargetY:              126,
		NextObsDistanceToTarget:             253,
		NextObsProjectileCount:              2,
		NextObsShooterWeaponReady:           0,
		NextObsShooterCooldownRemaining:     8,
		NextObsShooterHasActiveFireOrder:    1,
		NextObsShooterHasQueuedFireOrder:    0,
		NextObsShooterHasActiveMoveOrder:    1,
		NextObsShooterHasQueuedMoveOrder:    0,
		NextObsShooterHasDestination:        1,
		NextObsShooterDestinationX:          -100,
		NextObsShooterDestinationY:          40,
		NextObsShooterDistanceToDestination: 108,
		NextObsShooterRecentMoveFailure:     0,
		NextObsLocalTerrainPatch:            nextTerrainPatch,
		NextObsLocalOccupancyPatch:          nextOccupancyPatch,
		NextObsNearestFriendlyShotExists:    0,
		NextObsNearestFriendlyShotX:         0,
		NextObsNearestFriendlyShotY:         0,
		NextObsNearestFriendlyShotDist:      0,
		NextObsNearestHostileShotExists:     1,
		NextObsNearestHostileShotX:          -24,
		NextObsNearestHostileShotY:          16,
		NextObsNearestHostileShotDist:       32,
	}
}
