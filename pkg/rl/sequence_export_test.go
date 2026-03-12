package rl

import (
	"errors"
	"testing"
)

func TestNormalizedTrainingTransitionSequenceQueryClampsNegativeControls(t *testing.T) {
	normalized := normalizedTrainingTransitionSequenceQuery(TrainingTransitionSequenceQuery{
		TransitionQuery: TrainingTransitionQuery{
			EpisodeIDMin: 9,
			EpisodeIDMax: 2,
			Limit:        -4,
		},
		EpisodeLimit:      -2,
		MaxSequenceLength: -8,
	})

	if normalized.TransitionQuery.EpisodeIDMin != 2 || normalized.TransitionQuery.EpisodeIDMax != 9 {
		t.Fatalf("normalized episode bounds = [%d,%d], want [2,9]", normalized.TransitionQuery.EpisodeIDMin, normalized.TransitionQuery.EpisodeIDMax)
	}
	if normalized.TransitionQuery.Limit != 0 {
		t.Fatalf("normalized row limit = %d, want 0", normalized.TransitionQuery.Limit)
	}
	if normalized.EpisodeLimit != 0 {
		t.Fatalf("normalized episode limit = %d, want 0", normalized.EpisodeLimit)
	}
	if normalized.MaxSequenceLength != 0 {
		t.Fatalf("normalized max sequence length = %d, want 0", normalized.MaxSequenceLength)
	}
}

func TestTransitionSequenceStreamBuilderGroupsEpisodesAndSplitsWindows(t *testing.T) {
	var collected []TrainingTransitionSequence
	builder := newTransitionSequenceStreamBuilder(
		TrainingTransitionSequenceQuery{MaxSequenceLength: 2},
		func(sequence TrainingTransitionSequence) error {
			collected = append(collected, sequence)
			return nil
		},
	)

	records := []TrainingTransitionRecord{
		testTrainingTransitionRecord(11, 1),
		testTrainingTransitionRecord(11, 2),
		testTrainingTransitionRecord(11, 3),
		testTrainingTransitionRecord(12, 1),
	}
	for _, record := range records {
		if err := builder.Append(record); err != nil {
			t.Fatalf("Append(%d,%d) error = %v", record.EpisodeID, record.Tick, err)
		}
	}
	if err := builder.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if got, want := len(collected), 3; got != want {
		t.Fatalf("len(collected) = %d, want %d", got, want)
	}
	if collected[0].EpisodeID != 11 || collected[0].SequenceIndex != 0 || collected[0].StartTick != 1 || collected[0].EndTick != 2 || collected[0].StepCount != 2 {
		t.Fatalf("sequence[0] = %#v, want episode 11 window 0 ticks [1,2] count 2", collected[0])
	}
	if collected[1].EpisodeID != 11 || collected[1].SequenceIndex != 1 || collected[1].StartTick != 3 || collected[1].EndTick != 3 || collected[1].StepCount != 1 {
		t.Fatalf("sequence[1] = %#v, want episode 11 window 1 tick [3,3] count 1", collected[1])
	}
	if collected[2].EpisodeID != 12 || collected[2].SequenceIndex != 0 || collected[2].StartTick != 1 || collected[2].EndTick != 1 || collected[2].StepCount != 1 {
		t.Fatalf("sequence[2] = %#v, want episode 12 window 0 tick [1,1] count 1", collected[2])
	}
	if got, want := builder.sequencesExported, 3; got != want {
		t.Fatalf("builder.sequencesExported = %d, want %d", got, want)
	}
	if got, want := builder.episodesStarted, 2; got != want {
		t.Fatalf("builder.episodesStarted = %d, want %d", got, want)
	}
}

func TestTransitionSequenceStreamBuilderHonorsEpisodeLimit(t *testing.T) {
	var collected []TrainingTransitionSequence
	builder := newTransitionSequenceStreamBuilder(
		TrainingTransitionSequenceQuery{EpisodeLimit: 1},
		func(sequence TrainingTransitionSequence) error {
			collected = append(collected, sequence)
			return nil
		},
	)

	if err := builder.Append(testTrainingTransitionRecord(21, 1)); err != nil {
		t.Fatalf("Append(first episode) error = %v", err)
	}
	err := builder.Append(testTrainingTransitionRecord(22, 1))
	if !errors.Is(err, errTransitionSequenceStreamStopped) {
		t.Fatalf("Append(second episode) error = %v, want errTransitionSequenceStreamStopped", err)
	}
	if err := builder.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if got, want := len(collected), 1; got != want {
		t.Fatalf("len(collected) = %d, want %d", got, want)
	}
	if collected[0].EpisodeID != 21 {
		t.Fatalf("collected[0].EpisodeID = %d, want 21", collected[0].EpisodeID)
	}
}

func testTrainingTransitionRecord(episodeID uint64, tick uint32) TrainingTransitionRecord {
	record := sampleTrainingTransitionRecord()
	record.EpisodeID = episodeID
	record.Tick = tick
	record.Scenario = DuelScenarioOpen
	record.Outcome = "target_killed"
	return record
}
