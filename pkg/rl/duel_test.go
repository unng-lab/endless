package rl

import (
	"context"
	"testing"
)

type memoryRecorder struct {
	steps    []StepRecord
	events   []EventRecord
	episodes []EpisodeRecord
}

func (r *memoryRecorder) RecordStep(_ context.Context, step StepRecord) error {
	r.steps = append(r.steps, step)
	return nil
}

func (r *memoryRecorder) RecordEvents(_ context.Context, events []EventRecord) error {
	r.events = append(r.events, events...)
	return nil
}

func (r *memoryRecorder) RecordEpisode(_ context.Context, episode EpisodeRecord) error {
	r.episodes = append(r.episodes, episode)
	return nil
}

func (r *memoryRecorder) Close(context.Context) error {
	return nil
}

func TestDuelEnvironmentObserveReflectsQueuedMoveOrder(t *testing.T) {
	config := DuelRunConfig{
		Episodes:           1,
		MaxTicksPerEpisode: 60,
		Seed:               1,
		WorldColumns:       64,
		WorldRows:          64,
		TileSize:           16,
	}
	environment := NewDuelEnvironment(config)
	defer environment.Close()

	observation, err := environment.Reset(11)
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	action := Action{
		Type: ActionTypeMove,
		MoveTarget: clampPointToWorld(
			observation.Snapshot.Shooter.Position,
			observation.WorldWidth,
			observation.WorldHeight,
		),
	}
	action.MoveTarget.X += config.TileSize * 3

	accepted, err := environment.ApplyAction(action)
	if err != nil {
		t.Fatalf("ApplyAction() error = %v", err)
	}
	if !accepted {
		t.Fatal("ApplyAction() accepted = false, want true")
	}

	afterApply, err := environment.Observe()
	if err != nil {
		t.Fatalf("Observe() after ApplyAction error = %v", err)
	}
	if !afterApply.Snapshot.Shooter.HasQueuedMoveOrder {
		t.Fatal("expected queued move order to be visible in observation before the next step")
	}
	expectedPatchArea := (duelObservationPatchRadius*2 + 1) * (duelObservationPatchRadius*2 + 1)
	if len(afterApply.LocalTerrainPatch) != expectedPatchArea {
		t.Fatalf("LocalTerrainPatch len = %d, want %d", len(afterApply.LocalTerrainPatch), expectedPatchArea)
	}
	if len(afterApply.LocalOccupancyPatch) != expectedPatchArea {
		t.Fatalf("LocalOccupancyPatch len = %d, want %d", len(afterApply.LocalOccupancyPatch), expectedPatchArea)
	}

	centerIndex := expectedPatchArea / 2
	if afterApply.LocalOccupancyPatch[centerIndex] != occupancyShooter {
		t.Fatalf("center occupancy = %d, want shooter code %d", afterApply.LocalOccupancyPatch[centerIndex], occupancyShooter)
	}
}

func TestDuelEnvironmentStepBuildsShooterDestinationFeatures(t *testing.T) {
	config := DuelRunConfig{
		Episodes:           1,
		MaxTicksPerEpisode: 60,
		Seed:               1,
		WorldColumns:       64,
		WorldRows:          64,
		TileSize:           16,
		Scenario:           DuelScenarioOpen,
	}
	environment := NewDuelEnvironment(config)
	defer environment.Close()

	observation, err := environment.Reset(31)
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	moveTarget := observation.Snapshot.Shooter.Position
	moveTarget.X += config.TileSize * 3
	if _, err := environment.ApplyAction(Action{Type: ActionTypeMove, MoveTarget: moveTarget}); err != nil {
		t.Fatalf("ApplyAction(move) error = %v", err)
	}

	stepResult, err := environment.Step()
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if !stepResult.After.ShooterHasDestination {
		t.Fatal("expected shooter destination features after move order starts")
	}
	if stepResult.After.ShooterDistanceToDestination <= 0 {
		t.Fatalf("ShooterDistanceToDestination = %f, want > 0", stepResult.After.ShooterDistanceToDestination)
	}
	if stepResult.After.ShooterRecentMoveFailure {
		t.Fatal("expected successful move start to keep recent move failure false")
	}
}

func TestDuelEnvironmentObservationReportsNearestFriendlyProjectile(t *testing.T) {
	config := DuelRunConfig{
		Episodes:           1,
		MaxTicksPerEpisode: 60,
		Seed:               1,
		WorldColumns:       64,
		WorldRows:          64,
		TileSize:           16,
	}
	environment := NewDuelEnvironment(config)
	defer environment.Close()

	observation, err := environment.Reset(17)
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	fireDirection := observation.Snapshot.RelativeTarget
	accepted, err := environment.ApplyAction(Action{
		Type:          ActionTypeFire,
		FireDirection: fireDirection,
	})
	if err != nil {
		t.Fatalf("ApplyAction(fire) error = %v", err)
	}
	if !accepted {
		t.Fatal("ApplyAction(fire) accepted = false, want true")
	}

	foundFriendlyProjectile := false
	for stepIndex := 0; stepIndex < 16; stepIndex++ {
		stepResult, err := environment.Step()
		if err != nil {
			t.Fatalf("Step() error = %v", err)
		}
		if stepResult.After.NearestFriendlyShot.Exists {
			foundFriendlyProjectile = true
			break
		}
	}

	if !foundFriendlyProjectile {
		t.Fatal("expected nearest friendly projectile feature after firing")
	}
}

func TestDuelEnvironmentWithCoverReportsMovementBlockersInLocalPatch(t *testing.T) {
	config := DuelRunConfig{
		Episodes:           1,
		MaxTicksPerEpisode: 60,
		Seed:               1,
		WorldColumns:       64,
		WorldRows:          64,
		TileSize:           16,
		Scenario:           DuelScenarioWithCover,
	}
	environment := NewDuelEnvironment(config)
	defer environment.Close()

	observation, err := environment.Reset(23)
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	foundBlocker := false
	for _, occupancy := range observation.LocalOccupancyPatch {
		if occupancy == occupancyMovementBlocker {
			foundBlocker = true
			break
		}
	}
	if !foundBlocker {
		t.Fatal("expected local occupancy patch to include movement blockers in cover scenario")
	}
}

func TestDuelEnvironmentReportsRecentMoveFailureAfterBlockedDestination(t *testing.T) {
	config := DuelRunConfig{
		Episodes:           1,
		MaxTicksPerEpisode: 60,
		Seed:               1,
		WorldColumns:       64,
		WorldRows:          64,
		TileSize:           16,
		Scenario:           DuelScenarioWithCover,
	}
	environment := NewDuelEnvironment(config)
	defer environment.Close()

	observation, err := environment.Reset(37)
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	blockedTarget := observation.Snapshot.Shooter.Position
	blockedTarget.X += config.TileSize * 2
	accepted, err := environment.ApplyAction(Action{Type: ActionTypeMove, MoveTarget: blockedTarget})
	if err == nil {
		t.Fatal("ApplyAction(blocked move) error = nil, want path failure")
	}
	if accepted {
		t.Fatal("ApplyAction(blocked move) accepted = true, want false")
	}

	stepResult, err := environment.Step()
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if !stepResult.After.ShooterRecentMoveFailure {
		t.Fatal("expected recent move failure after blocked move action")
	}
}

func TestRunDuelCollectionRecordsMoveAndFireTransitions(t *testing.T) {
	recorder := &memoryRecorder{}
	config := DuelRunConfig{
		Episodes:           1,
		MaxTicksPerEpisode: 320,
		Seed:               5,
		WorldColumns:       64,
		WorldRows:          64,
		TileSize:           16,
	}

	if err := RunDuelCollection(context.Background(), config, recorder); err != nil {
		t.Fatalf("RunDuelCollection() error = %v", err)
	}
	if len(recorder.episodes) != 1 {
		t.Fatalf("episodes recorded = %d, want 1", len(recorder.episodes))
	}
	if len(recorder.steps) == 0 {
		t.Fatal("expected at least one recorded step")
	}
	if len(recorder.events) == 0 {
		t.Fatal("expected sparse events to be recorded alongside steps")
	}

	hasMove := false
	hasAcceptedMove := false
	hasFire := false
	hasPreObservation := false
	hasTerrainPatch := false
	hasProjectileFeature := false
	for _, step := range recorder.steps {
		if step.ObsDistanceToTarget > 0 {
			hasPreObservation = true
		}
		if expectedPatchArea := (int(step.ObsPatchRadius)*2 + 1) * (int(step.ObsPatchRadius)*2 + 1); expectedPatchArea > 0 && len(step.ObsLocalTerrainPatch) == expectedPatchArea && len(step.LocalTerrainPatch) == expectedPatchArea {
			hasTerrainPatch = true
		}
		if step.NearestFriendlyShotExists == 1 || step.ObsNearestFriendlyShotExists == 1 {
			hasProjectileFeature = true
		}
		if step.ActionType == string(ActionTypeMove) {
			hasMove = true
			if step.ActionAccepted == 1 {
				hasAcceptedMove = true
			}
		}
		if step.ActionType == string(ActionTypeFire) {
			hasFire = true
		}
	}

	if !hasPreObservation {
		t.Fatal("expected step rows to carry pre-action observation fields")
	}
	if !hasMove {
		t.Fatal("expected dataset to include move actions")
	}
	if !hasAcceptedMove {
		t.Fatal("expected at least one accepted move action")
	}
	if !hasFire {
		t.Fatal("expected dataset to include fire actions")
	}
	if !hasTerrainPatch {
		t.Fatal("expected step rows to include local terrain patches")
	}
	if !hasProjectileFeature {
		t.Fatal("expected step rows to include projectile features")
	}
}

func TestRunDuelCollectionWithCoverScenarioRecordsConfiguredScenarioName(t *testing.T) {
	recorder := &memoryRecorder{}
	config := DuelRunConfig{
		Episodes:           1,
		MaxTicksPerEpisode: 120,
		Seed:               13,
		WorldColumns:       64,
		WorldRows:          64,
		TileSize:           16,
		Scenario:           DuelScenarioWithCover,
	}

	if err := RunDuelCollection(context.Background(), config, recorder); err != nil {
		t.Fatalf("RunDuelCollection() error = %v", err)
	}
	if len(recorder.episodes) != 1 {
		t.Fatalf("episodes recorded = %d, want 1", len(recorder.episodes))
	}
	if recorder.episodes[0].Scenario != DuelScenarioWithCover {
		t.Fatalf("episode scenario = %q, want %q", recorder.episodes[0].Scenario, DuelScenarioWithCover)
	}
}

func TestNewPolicyByNameBuildsSupportedPolicies(t *testing.T) {
	if _, err := NewPolicyByName(PolicyLeadAndStrafe, 1); err != nil {
		t.Fatalf("NewPolicyByName(%q) error = %v", PolicyLeadAndStrafe, err)
	}
	if _, err := NewPolicyByName(PolicyRandom, 1); err != nil {
		t.Fatalf("NewPolicyByName(%q) error = %v", PolicyRandom, err)
	}
	if _, err := NewPolicyByName("unsupported", 1); err == nil {
		t.Fatal("NewPolicyByName() error = nil, want unsupported-policy error")
	}
}

func TestRunDuelEvaluationAggregatesEpisodes(t *testing.T) {
	policy, err := NewPolicyByName(PolicyRandom, 7)
	if err != nil {
		t.Fatalf("NewPolicyByName() error = %v", err)
	}

	summary, err := RunDuelEvaluation(context.Background(), DuelRunConfig{
		Episodes:           3,
		MaxTicksPerEpisode: 120,
		Seed:               7,
		WorldColumns:       64,
		WorldRows:          64,
		TileSize:           16,
	}, policy)
	if err != nil {
		t.Fatalf("RunDuelEvaluation() error = %v", err)
	}

	if summary.EpisodesGenerated != 3 {
		t.Fatalf("EpisodesGenerated = %d, want 3", summary.EpisodesGenerated)
	}
	if summary.AverageTicks <= 0 {
		t.Fatalf("AverageTicks = %f, want > 0", summary.AverageTicks)
	}
	if summary.ShotsFired < 0 || summary.ProjectileHits < 0 || summary.ProjectileExpired < 0 {
		t.Fatalf("invalid negative combat counters: %+v", summary)
	}
}
