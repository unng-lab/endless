package rl

import (
	"context"
	"fmt"
	"math"
)

const (
	defaultLinearQStubEpochs         = 5
	defaultLinearQStubBatchSize      = 64
	defaultLinearQStubLearningRate   = 0.005
	defaultLinearQStubDiscount       = 0.99
	defaultLinearQStubTargetAbsMax   = 16
	defaultLinearQStubResidualAbsMax = 8
)

// VectorizedTransitionDataset keeps the in-memory transition slice together with the tensor
// inspection summary gathered while vectorizing rows from the trainer-facing ClickHouse view.
type VectorizedTransitionDataset struct {
	Transitions []VectorizedTransition
	Inspection  TransitionTensorInspection
}

// LinearQStubConfig defines the first Go-side learner stub settings. The stub intentionally
// stays simple: it trains one linear critic over the frozen observation/action tensors so the
// CLI can smoke-check an end-to-end offline training pass without introducing a long-lived
// inference/runtime contract inside gameplay code.
type LinearQStubConfig struct {
	Epochs       int
	BatchSize    int
	LearningRate float32
	Discount     float32
}

// LinearQStubModel stores the minimal trainable parameters for the Go-side learner stub.
// The model scores one observation-action pair with a single linear value estimate.
type LinearQStubModel struct {
	ObsDim    int
	ActionDim int
	Weights   []float32
	Bias      float32
}

// LinearQStubEpochSummary reports one post-epoch evaluation pass so CLI tools can see whether
// the stub is converging or diverging on the currently selected offline transition slice.
type LinearQStubEpochSummary struct {
	Epoch               int
	AverageLoss         float32
	MeanAbsoluteTDError float32
	AveragePrediction   float32
	AverageTarget       float32
	LinkedNextActions   int
	TerminalTransitions int
	UnlinkedTransitions int
}

// LinearQStubTrainingSummary aggregates the dataset shape and the training metrics that are
// most useful when validating the first offline learner loop from the command line.
type LinearQStubTrainingSummary struct {
	Samples                int
	ObsDim                 int
	ActionDim              int
	InputDim               int
	Epochs                 int
	BatchSize              int
	LinkedNextActions      int
	TerminalTransitions    int
	UnlinkedTransitions    int
	InitialAverageLoss     float32
	FinalAverageLoss       float32
	FinalMeanAbsoluteTD    float32
	FinalAveragePrediction float32
	FinalAverageTarget     float32
	PredictionMin          float32
	PredictionMax          float32
	TargetMin              float32
	TargetMax              float32
	EpochSummaries         []LinearQStubEpochSummary
}

type linearQStubExample struct {
	Transition       VectorizedTransition
	NextAction       []float32
	HasNextAction    bool
	DiscountedReturn float32
}

type linearQStubEvaluationMetrics struct {
	AverageLoss         float32
	MeanAbsoluteTDError float32
	AveragePrediction   float32
	AverageTarget       float32
	PredictionMin       float32
	PredictionMax       float32
	TargetMin           float32
	TargetMax           float32
}

// DefaultLinearQStubConfig returns conservative defaults for the first offline learner smoke
// test so callers can override only the few parameters they care about from the CLI.
func DefaultLinearQStubConfig() LinearQStubConfig {
	return LinearQStubConfig{
		Epochs:       defaultLinearQStubEpochs,
		BatchSize:    defaultLinearQStubBatchSize,
		LearningRate: defaultLinearQStubLearningRate,
		Discount:     defaultLinearQStubDiscount,
	}
}

// Normalized fills omitted learner parameters and clamps the discount into the standard
// [0,1] interval so the linear stub does not silently run with nonsensical values.
func (c LinearQStubConfig) Normalized() LinearQStubConfig {
	defaults := DefaultLinearQStubConfig()
	if c.Epochs <= 0 {
		c.Epochs = defaults.Epochs
	}
	if c.BatchSize <= 0 {
		c.BatchSize = defaults.BatchSize
	}
	if c.LearningRate <= 0 {
		c.LearningRate = defaults.LearningRate
	}
	c.Discount = clampFloat32(c.Discount, 0, 1)
	return c
}

// LoadVectorizedTransitionDataset streams raw trainer-facing rows from ClickHouse and eagerly
// vectorizes them into the frozen tensor contract. The learner stub uses this in-memory dataset
// because it may need multiple deterministic epochs over the same selected transition slice.
func LoadVectorizedTransitionDataset(ctx context.Context, reader *ClickHouseTransitionReader, query TrainingTransitionQuery, spec TransitionNormalizationSpec) (VectorizedTransitionDataset, error) {
	if reader == nil {
		return VectorizedTransitionDataset{}, fmt.Errorf("clickhouse transition reader is nil")
	}

	spec = spec.Normalized()
	dataset := VectorizedTransitionDataset{}
	_, err := reader.StreamTransitions(ctx, query, func(record TrainingTransitionRecord) error {
		transition, err := VectorizeTransition(record, spec)
		if err != nil {
			return err
		}
		dataset.Transitions = append(dataset.Transitions, transition)
		dataset.Inspection.ObserveTransition(transition)
		return nil
	})
	if err != nil {
		return VectorizedTransitionDataset{}, err
	}
	return dataset, nil
}

// NewLinearQStubModel allocates one zero-initialized linear critic with the dimensions implied
// by the frozen observation and action tensor contract.
func NewLinearQStubModel(obsDim, actionDim int) (LinearQStubModel, error) {
	if obsDim <= 0 {
		return LinearQStubModel{}, fmt.Errorf("observation dim must be positive")
	}
	if actionDim <= 0 {
		return LinearQStubModel{}, fmt.Errorf("action dim must be positive")
	}
	return LinearQStubModel{
		ObsDim:    obsDim,
		ActionDim: actionDim,
		Weights:   make([]float32, obsDim+actionDim),
	}, nil
}

// Predict scores one observation-action pair with the current linear critic parameters.
func (m LinearQStubModel) Predict(obs, action []float32) (float32, error) {
	if len(obs) != m.ObsDim {
		return 0, fmt.Errorf("observation dim = %d, want %d", len(obs), m.ObsDim)
	}
	if len(action) != m.ActionDim {
		return 0, fmt.Errorf("action dim = %d, want %d", len(action), m.ActionDim)
	}

	value := m.Bias
	for index, feature := range obs {
		value += m.Weights[index] * feature
	}
	for index, feature := range action {
		value += m.Weights[m.ObsDim+index] * feature
	}
	if !isFiniteFloat32(value) {
		return 0, fmt.Errorf("prediction became non-finite")
	}
	return value, nil
}

// TrainLinearQStub runs the first Go-side learner stub over already vectorized transitions.
// The implementation intentionally remains small and explicit: it links consecutive rows from
// the same episode, computes discounted return-to-go targets and performs batch-averaged
// regression updates so operators can verify that offline training behaves sensibly from the CLI.
func TrainLinearQStub(ctx context.Context, transitions []VectorizedTransition, config LinearQStubConfig) (LinearQStubModel, LinearQStubTrainingSummary, error) {
	if len(transitions) == 0 {
		return LinearQStubModel{}, LinearQStubTrainingSummary{}, fmt.Errorf("transition dataset is empty")
	}

	config = config.Normalized()
	obsDim := len(transitions[0].Obs)
	actionDim := len(transitions[0].Action)
	for index, transition := range transitions {
		if len(transition.Obs) != obsDim {
			return LinearQStubModel{}, LinearQStubTrainingSummary{}, fmt.Errorf("transition %d observation dim = %d, want %d", index, len(transition.Obs), obsDim)
		}
		if len(transition.Action) != actionDim {
			return LinearQStubModel{}, LinearQStubTrainingSummary{}, fmt.Errorf("transition %d action dim = %d, want %d", index, len(transition.Action), actionDim)
		}
		if len(transition.NextObs) != obsDim {
			return LinearQStubModel{}, LinearQStubTrainingSummary{}, fmt.Errorf("transition %d next observation dim = %d, want %d", index, len(transition.NextObs), obsDim)
		}
	}

	model, err := NewLinearQStubModel(obsDim, actionDim)
	if err != nil {
		return LinearQStubModel{}, LinearQStubTrainingSummary{}, err
	}

	examples, linkedNextActions, terminalTransitions, err := buildLinearQStubExamples(transitions, actionDim)
	if err != nil {
		return LinearQStubModel{}, LinearQStubTrainingSummary{}, err
	}
	annotateLinearQStubReturns(examples, config.Discount)

	initialMetrics, err := evaluateLinearQStubModel(model, examples)
	if err != nil {
		return LinearQStubModel{}, LinearQStubTrainingSummary{}, err
	}

	summary := LinearQStubTrainingSummary{
		Samples:             len(examples),
		ObsDim:              obsDim,
		ActionDim:           actionDim,
		InputDim:            obsDim + actionDim,
		Epochs:              config.Epochs,
		BatchSize:           config.BatchSize,
		LinkedNextActions:   linkedNextActions,
		TerminalTransitions: terminalTransitions,
		UnlinkedTransitions: len(examples) - linkedNextActions - terminalTransitions,
		InitialAverageLoss:  initialMetrics.AverageLoss,
	}

	for epoch := 1; epoch <= config.Epochs; epoch++ {
		if err := trainLinearQStubEpoch(ctx, &model, examples, config); err != nil {
			return LinearQStubModel{}, LinearQStubTrainingSummary{}, err
		}

		metrics, err := evaluateLinearQStubModel(model, examples)
		if err != nil {
			return LinearQStubModel{}, LinearQStubTrainingSummary{}, err
		}
		summary.EpochSummaries = append(summary.EpochSummaries, LinearQStubEpochSummary{
			Epoch:               epoch,
			AverageLoss:         metrics.AverageLoss,
			MeanAbsoluteTDError: metrics.MeanAbsoluteTDError,
			AveragePrediction:   metrics.AveragePrediction,
			AverageTarget:       metrics.AverageTarget,
			LinkedNextActions:   summary.LinkedNextActions,
			TerminalTransitions: summary.TerminalTransitions,
			UnlinkedTransitions: summary.UnlinkedTransitions,
		})
	}

	finalMetrics := initialMetrics
	if len(summary.EpochSummaries) > 0 {
		finalMetrics, err = evaluateLinearQStubModel(model, examples)
		if err != nil {
			return LinearQStubModel{}, LinearQStubTrainingSummary{}, err
		}
	}

	summary.FinalAverageLoss = finalMetrics.AverageLoss
	summary.FinalMeanAbsoluteTD = finalMetrics.MeanAbsoluteTDError
	summary.FinalAveragePrediction = finalMetrics.AveragePrediction
	summary.FinalAverageTarget = finalMetrics.AverageTarget
	summary.PredictionMin = finalMetrics.PredictionMin
	summary.PredictionMax = finalMetrics.PredictionMax
	summary.TargetMin = finalMetrics.TargetMin
	summary.TargetMax = finalMetrics.TargetMax

	return model, summary, nil
}

// buildLinearQStubExamples reconstructs the next action linkage and episode continuity metadata
// from one deterministic transition slice so later training code may derive per-row targets.
func buildLinearQStubExamples(transitions []VectorizedTransition, actionDim int) ([]linearQStubExample, int, int, error) {
	examples := make([]linearQStubExample, 0, len(transitions))
	linkedNextActions := 0
	terminalTransitions := 0
	for index := range transitions {
		transition := transitions[index]
		if len(transition.Action) != actionDim {
			return nil, 0, 0, fmt.Errorf("transition %d action dim = %d, want %d", index, len(transition.Action), actionDim)
		}

		example := linearQStubExample{Transition: transition}
		if transition.Done >= 0.5 {
			terminalTransitions++
			examples = append(examples, example)
			continue
		}
		if index+1 < len(transitions) &&
			transitions[index+1].EpisodeID == transition.EpisodeID &&
			transitions[index+1].Tick == transition.Tick+1 {
			example.NextAction = append([]float32(nil), transitions[index+1].Action...)
			example.HasNextAction = true
			linkedNextActions++
		}
		examples = append(examples, example)
	}
	return examples, linkedNextActions, terminalTransitions, nil
}

// annotateLinearQStubReturns computes one discounted return-to-go target per example by
// walking every recorded episode backwards. Unlinked rows fall back to their immediate reward.
func annotateLinearQStubReturns(examples []linearQStubExample, discount float32) {
	if len(examples) == 0 {
		return
	}

	discount = clampFloat32(discount, 0, 1)
	for index := len(examples) - 1; index >= 0; index-- {
		examples[index].DiscountedReturn = clampLinearQStubTarget(examples[index].Transition.Reward)
		if examples[index].Transition.Done >= 0.5 {
			continue
		}
		if index+1 >= len(examples) {
			continue
		}
		nextExample := examples[index+1]
		if nextExample.Transition.EpisodeID != examples[index].Transition.EpisodeID {
			continue
		}
		if nextExample.Transition.Tick != examples[index].Transition.Tick+1 {
			continue
		}
		examples[index].DiscountedReturn = clampLinearQStubTarget(examples[index].DiscountedReturn + discount*nextExample.DiscountedReturn)
	}
}

// trainLinearQStubEpoch applies one pass of batch-averaged semi-gradient updates so the stub
// can learn from fixed-size slices without introducing any external optimizer dependency.
func trainLinearQStubEpoch(ctx context.Context, model *LinearQStubModel, examples []linearQStubExample, config LinearQStubConfig) error {
	if model == nil {
		return fmt.Errorf("linear q stub model is nil")
	}

	inputDim := model.ObsDim + model.ActionDim
	gradient := make([]float32, inputDim)
	for batchStart := 0; batchStart < len(examples); batchStart += config.BatchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		batchEnd := batchStart + config.BatchSize
		if batchEnd > len(examples) {
			batchEnd = len(examples)
		}
		for index := range gradient {
			gradient[index] = 0
		}
		biasGradient := float32(0)

		for _, example := range examples[batchStart:batchEnd] {
			prediction, err := model.Predict(example.Transition.Obs, example.Transition.Action)
			if err != nil {
				return err
			}
			residual := clampFloat32(example.DiscountedReturn-prediction, -defaultLinearQStubResidualAbsMax, defaultLinearQStubResidualAbsMax)
			for index, feature := range example.Transition.Obs {
				gradient[index] += residual * feature
			}
			for index, feature := range example.Transition.Action {
				gradient[model.ObsDim+index] += residual * feature
			}
			biasGradient += residual
		}

		stepScale := config.LearningRate / float32(batchEnd-batchStart)
		for index := range model.Weights {
			model.Weights[index] += stepScale * gradient[index]
			if !isFiniteFloat32(model.Weights[index]) {
				return fmt.Errorf("weight %d became non-finite", index)
			}
		}
		model.Bias += stepScale * biasGradient
		if !isFiniteFloat32(model.Bias) {
			return fmt.Errorf("bias became non-finite")
		}
	}
	return nil
}

// evaluateLinearQStubModel replays the current dataset without mutating weights so callers can
// compare epochs by one stable set of aggregate residual statistics against frozen targets.
func evaluateLinearQStubModel(model LinearQStubModel, examples []linearQStubExample) (linearQStubEvaluationMetrics, error) {
	if len(examples) == 0 {
		return linearQStubEvaluationMetrics{}, fmt.Errorf("linear q stub examples are empty")
	}

	metrics := linearQStubEvaluationMetrics{}
	first := true
	for _, example := range examples {
		prediction, err := model.Predict(example.Transition.Obs, example.Transition.Action)
		if err != nil {
			return linearQStubEvaluationMetrics{}, err
		}
		target := example.DiscountedReturn
		if !isFiniteFloat32(target) {
			return linearQStubEvaluationMetrics{}, fmt.Errorf("target became non-finite")
		}

		tdError := target - prediction
		loss := tdError * tdError
		metrics.AverageLoss += loss
		if tdError < 0 {
			metrics.MeanAbsoluteTDError -= tdError
		} else {
			metrics.MeanAbsoluteTDError += tdError
		}
		metrics.AveragePrediction += prediction
		metrics.AverageTarget += target
		if first {
			metrics.PredictionMin = prediction
			metrics.PredictionMax = prediction
			metrics.TargetMin = target
			metrics.TargetMax = target
			first = false
		} else {
			metrics.PredictionMin = minFloat32(metrics.PredictionMin, prediction)
			metrics.PredictionMax = maxFloat32(metrics.PredictionMax, prediction)
			metrics.TargetMin = minFloat32(metrics.TargetMin, target)
			metrics.TargetMax = maxFloat32(metrics.TargetMax, target)
		}
	}

	sampleCount := float32(len(examples))
	metrics.AverageLoss /= sampleCount
	metrics.MeanAbsoluteTDError /= sampleCount
	metrics.AveragePrediction /= sampleCount
	metrics.AverageTarget /= sampleCount
	return metrics, nil
}

func clampLinearQStubTarget(value float32) float32 {
	return clampFloat32(value, -defaultLinearQStubTargetAbsMax, defaultLinearQStubTargetAbsMax)
}

func isFiniteFloat32(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}
