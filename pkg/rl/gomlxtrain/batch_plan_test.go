package gomlxtrain

import "testing"

func TestResolveTrainingBatchPlanDropsIncompleteTailBatches(t *testing.T) {
	t.Parallel()

	plan, err := resolveTrainingBatchPlan(1000, 128)
	if err != nil {
		t.Fatalf("resolveTrainingBatchPlan() error = %v", err)
	}

	if got, want := plan.BatchSize, 128; got != want {
		t.Fatalf("resolveTrainingBatchPlan() batch size = %d, want %d", got, want)
	}
	if got, want := plan.StepsPerEpoch, 7; got != want {
		t.Fatalf("resolveTrainingBatchPlan() steps per epoch = %d, want %d", got, want)
	}
	if !plan.DropIncompleteBatch {
		t.Fatal("resolveTrainingBatchPlan() drop incomplete batch = false, want true")
	}
}

func TestResolveTrainingBatchPlanClampsOversizedBatchToDataset(t *testing.T) {
	t.Parallel()

	plan, err := resolveTrainingBatchPlan(64, 128)
	if err != nil {
		t.Fatalf("resolveTrainingBatchPlan() error = %v", err)
	}

	if got, want := plan.BatchSize, 64; got != want {
		t.Fatalf("resolveTrainingBatchPlan() batch size = %d, want %d", got, want)
	}
	if got, want := plan.StepsPerEpoch, 1; got != want {
		t.Fatalf("resolveTrainingBatchPlan() steps per epoch = %d, want %d", got, want)
	}
}

func TestResolveTrainingBatchPlanRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	if _, err := resolveTrainingBatchPlan(0, 128); err == nil {
		t.Fatal("resolveTrainingBatchPlan(samples=0) error = nil, want validation error")
	}
	if _, err := resolveTrainingBatchPlan(16, 0); err == nil {
		t.Fatal("resolveTrainingBatchPlan(batch=0) error = nil, want validation error")
	}
}
