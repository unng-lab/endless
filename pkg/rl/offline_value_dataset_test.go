package rl

import "testing"

func TestBuildDiscountedReturnExamplesBuildsReturnToGoAndSummary(t *testing.T) {
	transitions := []VectorizedTransition{
		{
			EpisodeID: 1,
			Tick:      1,
			Obs:       []float32{1},
			Action:    []float32{0},
			Reward:    1,
			Done:      0,
			NextObs:   []float32{2},
		},
		{
			EpisodeID: 1,
			Tick:      2,
			Obs:       []float32{2},
			Action:    []float32{1},
			Reward:    2,
			Done:      1,
			NextObs:   []float32{3},
		},
		{
			EpisodeID: 2,
			Tick:      1,
			Obs:       []float32{3},
			Action:    []float32{0},
			Reward:    4,
			Done:      0,
			NextObs:   []float32{4},
		},
	}

	examples, summary, err := BuildDiscountedReturnExamples(transitions, 0.5)
	if err != nil {
		t.Fatalf("BuildDiscountedReturnExamples() error = %v", err)
	}
	if got, want := len(examples), 3; got != want {
		t.Fatalf("len(examples) = %d, want %d", got, want)
	}
	if got, want := examples[0].DiscountedReturn, float32(2); got != want {
		t.Fatalf("examples[0].DiscountedReturn = %f, want %f", got, want)
	}
	if got, want := examples[1].DiscountedReturn, float32(2); got != want {
		t.Fatalf("examples[1].DiscountedReturn = %f, want %f", got, want)
	}
	if got, want := examples[2].DiscountedReturn, float32(4); got != want {
		t.Fatalf("examples[2].DiscountedReturn = %f, want %f", got, want)
	}
	if got, want := summary.ContinuedTransitions, 1; got != want {
		t.Fatalf("summary.ContinuedTransitions = %d, want %d", got, want)
	}
	if got, want := summary.TerminalTransitions, 1; got != want {
		t.Fatalf("summary.TerminalTransitions = %d, want %d", got, want)
	}
	if got, want := summary.UnlinkedTransitions, 1; got != want {
		t.Fatalf("summary.UnlinkedTransitions = %d, want %d", got, want)
	}
	if got, want := summary.DiscountedReturnMin, float32(2); got != want {
		t.Fatalf("summary.DiscountedReturnMin = %f, want %f", got, want)
	}
	if got, want := summary.DiscountedReturnMax, float32(4); got != want {
		t.Fatalf("summary.DiscountedReturnMax = %f, want %f", got, want)
	}
}
