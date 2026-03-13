package gomlxtrain

import "fmt"

// trainingBatchPlan describes one stable batching strategy for the GoMLX trainer.
// The training loop intentionally drops incomplete tail batches so the executor never
// has to rebuild the model graph for a second leading dimension at the end of each epoch.
type trainingBatchPlan struct {
	BatchSize           int
	StepsPerEpoch       int
	DropIncompleteBatch bool
}

// resolveTrainingBatchPlan selects one effective batch size for the prepared dataset.
// It clamps oversized operator input down to the dataset size so tiny datasets still
// execute at least one full batch, and it always drops remainder batches to keep the
// compiled GoMLX graph shape stable across all training steps.
func resolveTrainingBatchPlan(samples int, requestedBatchSize int) (trainingBatchPlan, error) {
	if samples <= 0 {
		return trainingBatchPlan{}, fmt.Errorf("samples must be positive, got %d", samples)
	}
	if requestedBatchSize <= 0 {
		return trainingBatchPlan{}, fmt.Errorf("batch size must be positive, got %d", requestedBatchSize)
	}

	effectiveBatchSize := requestedBatchSize
	if effectiveBatchSize > samples {
		effectiveBatchSize = samples
	}

	stepsPerEpoch := samples / effectiveBatchSize
	if stepsPerEpoch <= 0 {
		return trainingBatchPlan{}, fmt.Errorf(
			"steps per epoch resolved to zero for samples=%d batch_size=%d",
			samples,
			effectiveBatchSize,
		)
	}

	return trainingBatchPlan{
		BatchSize:           effectiveBatchSize,
		StepsPerEpoch:       stepsPerEpoch,
		DropIncompleteBatch: true,
	}, nil
}
