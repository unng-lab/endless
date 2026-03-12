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
	for _, step := range recorder.steps {
		if step.ObsDistanceToTarget > 0 {
			hasPreObservation = true
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
}
