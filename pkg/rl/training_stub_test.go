package rl

import (
	"context"
	"math"
	"testing"
)

func TestBuildLinearQStubExamplesLinksNextActionWithinEpisode(t *testing.T) {
	transitions := []VectorizedTransition{
		{
			EpisodeID: 1,
			Tick:      1,
			Obs:       []float32{0.1},
			Action:    []float32{0.2},
			Reward:    0.5,
			Done:      0,
			NextObs:   []float32{0.3},
		},
		{
			EpisodeID: 1,
			Tick:      2,
			Obs:       []float32{0.3},
			Action:    []float32{0.4},
			Reward:    1,
			Done:      1,
			NextObs:   []float32{0},
		},
		{
			EpisodeID: 2,
			Tick:      1,
			Obs:       []float32{0.5},
			Action:    []float32{0.6},
			Reward:    0.25,
			Done:      0,
			NextObs:   []float32{0.7},
		},
	}

	examples, linkedNextActions, terminalTransitions, err := buildLinearQStubExamples(transitions, 1)
	if err != nil {
		t.Fatalf("buildLinearQStubExamples() error = %v", err)
	}
	if got, want := len(examples), 3; got != want {
		t.Fatalf("len(examples) = %d, want %d", got, want)
	}
	if got, want := linkedNextActions, 1; got != want {
		t.Fatalf("linkedNextActions = %d, want %d", got, want)
	}
	if got, want := terminalTransitions, 1; got != want {
		t.Fatalf("terminalTransitions = %d, want %d", got, want)
	}
	if !examples[0].HasNextAction {
		t.Fatal("examples[0].HasNextAction = false, want true")
	}
	if got, want := len(examples[0].NextAction), 1; got != want {
		t.Fatalf("len(examples[0].NextAction) = %d, want %d", got, want)
	}
	if got, want := examples[0].NextAction[0], float32(0.4); got != want {
		t.Fatalf("examples[0].NextAction[0] = %f, want %f", got, want)
	}
	if examples[1].HasNextAction {
		t.Fatal("examples[1].HasNextAction = true, want false for terminal transition")
	}
	if examples[2].HasNextAction {
		t.Fatal("examples[2].HasNextAction = true, want false when next row belongs to another episode")
	}
}

func TestTrainLinearQStubReducesLossOnTerminalRewardRegression(t *testing.T) {
	transitions := []VectorizedTransition{
		{
			EpisodeID: 1,
			Tick:      1,
			Obs:       []float32{1},
			Action:    []float32{0},
			Reward:    1,
			Done:      1,
			NextObs:   []float32{0},
		},
		{
			EpisodeID: 2,
			Tick:      1,
			Obs:       []float32{0},
			Action:    []float32{1},
			Reward:    2,
			Done:      1,
			NextObs:   []float32{0},
		},
		{
			EpisodeID: 3,
			Tick:      1,
			Obs:       []float32{1},
			Action:    []float32{1},
			Reward:    3,
			Done:      1,
			NextObs:   []float32{0},
		},
	}

	model, summary, err := TrainLinearQStub(context.Background(), transitions, LinearQStubConfig{
		Epochs:       200,
		BatchSize:    3,
		LearningRate: 0.1,
		Discount:     0,
	})
	if err != nil {
		t.Fatalf("TrainLinearQStub() error = %v", err)
	}

	if summary.InitialAverageLoss <= summary.FinalAverageLoss {
		t.Fatalf("loss did not improve: initial=%f final=%f", summary.InitialAverageLoss, summary.FinalAverageLoss)
	}
	if got, want := summary.TerminalTransitions, 3; got != want {
		t.Fatalf("summary.TerminalTransitions = %d, want %d", got, want)
	}
	if got, want := summary.LinkedNextActions, 0; got != want {
		t.Fatalf("summary.LinkedNextActions = %d, want %d", got, want)
	}

	predictionOne, err := model.Predict([]float32{1}, []float32{0})
	if err != nil {
		t.Fatalf("Predict(sample 1) error = %v", err)
	}
	predictionTwo, err := model.Predict([]float32{0}, []float32{1})
	if err != nil {
		t.Fatalf("Predict(sample 2) error = %v", err)
	}
	predictionThree, err := model.Predict([]float32{1}, []float32{1})
	if err != nil {
		t.Fatalf("Predict(sample 3) error = %v", err)
	}

	if math.Abs(float64(predictionOne-1)) > 0.15 {
		t.Fatalf("predictionOne = %f, want close to 1", predictionOne)
	}
	if math.Abs(float64(predictionTwo-2)) > 0.15 {
		t.Fatalf("predictionTwo = %f, want close to 2", predictionTwo)
	}
	if math.Abs(float64(predictionThree-3)) > 0.15 {
		t.Fatalf("predictionThree = %f, want close to 3", predictionThree)
	}
}
