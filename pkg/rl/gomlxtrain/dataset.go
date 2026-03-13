package gomlxtrain

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/unng-lab/endless/pkg/rl"
)

const (
	defaultBatchSize      = 128
	defaultEpochs         = 20
	defaultLearningRate   = 0.001
	defaultDiscount       = 0.99
	defaultTargetAbsMax   = 16
	defaultCheckpointKeep = 3
)

var defaultHiddenDims = []int{256, 128}

// DefaultConfig returns one conservative GoMLX trainer configuration that matches the current
// duel-scale tensor contract and still fits comfortably into the first WSL2 experiments.
func DefaultConfig() Config {
	return Config{
		Source: InputSourceConfig{
			Format: InputFormatJSONL,
		},
		CheckpointDir:  "artifacts/gomlx_critic",
		CheckpointKeep: defaultCheckpointKeep,
		BatchSize:      defaultBatchSize,
		Epochs:         defaultEpochs,
		LearningRate:   defaultLearningRate,
		Discount:       defaultDiscount,
		TargetAbsMax:   defaultTargetAbsMax,
		HiddenDims:     append([]int(nil), defaultHiddenDims...),
	}
}

// Normalized fills omitted values and validates operator-facing trainer settings before any
// expensive dataset load or GoMLX backend initialization happens.
func (c Config) Normalized() (Config, error) {
	defaults := DefaultConfig()

	switch InputFormat(strings.ToLower(strings.TrimSpace(string(c.Source.Format)))) {
	case "", InputFormatJSONL:
		c.Source.Format = InputFormatJSONL
	case InputFormatClickHouse:
		c.Source.Format = InputFormatClickHouse
	default:
		return Config{}, fmt.Errorf("unsupported input format %q; use jsonl or clickhouse", c.Source.Format)
	}

	if c.CheckpointDir == "" {
		c.CheckpointDir = defaults.CheckpointDir
	}
	if c.CheckpointKeep <= 0 {
		c.CheckpointKeep = defaults.CheckpointKeep
	}
	if c.BatchSize <= 0 {
		c.BatchSize = defaults.BatchSize
	}
	if c.Epochs <= 0 {
		c.Epochs = defaults.Epochs
	}
	if c.LearningRate <= 0 {
		c.LearningRate = defaults.LearningRate
	}
	if c.Discount < 0 || c.Discount > 1 {
		c.Discount = clampFloat32(c.Discount, 0, 1)
	}
	if c.TargetAbsMax < 0 {
		c.TargetAbsMax = defaults.TargetAbsMax
	}
	if len(c.HiddenDims) == 0 {
		c.HiddenDims = append([]int(nil), defaults.HiddenDims...)
	} else {
		c.HiddenDims = append([]int(nil), c.HiddenDims...)
	}
	for _, hiddenDim := range c.HiddenDims {
		if hiddenDim <= 0 {
			return Config{}, fmt.Errorf("hidden dims must be positive, got %v", c.HiddenDims)
		}
	}
	if c.Source.Format == InputFormatJSONL && strings.TrimSpace(c.Source.Path) == "" {
		return Config{}, fmt.Errorf("jsonl input requires -input-path")
	}
	return c, nil
}

// PrepareDataset loads transition rows from the configured source, vectorizes them according to
// the stable RL tensor contract and assembles one contiguous `(obs||action) -> discounted_return`
// dataset for the external GoMLX critic.
func PrepareDataset(ctx context.Context, cfg Config) (PreparedDataset, error) {
	cfg, err := cfg.Normalized()
	if err != nil {
		return PreparedDataset{}, err
	}

	records, err := loadTrainingTransitionRecords(ctx, cfg.Source)
	if err != nil {
		return PreparedDataset{}, err
	}
	if len(records) == 0 {
		return PreparedDataset{}, fmt.Errorf("no training transitions matched the selected source/query")
	}

	spec := rl.DefaultTransitionNormalizationSpec()
	vectorized := make([]rl.VectorizedTransition, 0, len(records))
	for index, record := range records {
		transition, err := rl.VectorizeTransition(record, spec)
		if err != nil {
			return PreparedDataset{}, fmt.Errorf("vectorize transition %d (episode=%d tick=%d): %w", index, record.EpisodeID, record.Tick, err)
		}
		vectorized = append(vectorized, transition)
	}

	examples, summary, err := rl.BuildDiscountedReturnExamples(vectorized, cfg.Discount)
	if err != nil {
		return PreparedDataset{}, err
	}

	obsDim := spec.ObservationDim()
	actionDim := spec.ActionDim()
	inputDim := obsDim + actionDim
	inputs := make([]float32, 0, len(examples)*inputDim)
	targets := make([]float32, 0, len(examples))

	targetMin := float32(0)
	targetMax := float32(0)
	for index, example := range examples {
		if len(example.Transition.Obs) != obsDim {
			return PreparedDataset{}, fmt.Errorf("example %d observation dim = %d, want %d", index, len(example.Transition.Obs), obsDim)
		}
		if len(example.Transition.Action) != actionDim {
			return PreparedDataset{}, fmt.Errorf("example %d action dim = %d, want %d", index, len(example.Transition.Action), actionDim)
		}
		inputs = append(inputs, example.Transition.Obs...)
		inputs = append(inputs, example.Transition.Action...)

		target := example.DiscountedReturn
		if cfg.TargetAbsMax > 0 {
			target = clampFloat32(target, -cfg.TargetAbsMax, cfg.TargetAbsMax)
		}
		targets = append(targets, target)
		if index == 0 {
			targetMin = target
			targetMax = target
		} else {
			targetMin = minFloat32(targetMin, target)
			targetMax = maxFloat32(targetMax, target)
		}
	}

	return PreparedDataset{
		Inputs:                  inputs,
		Targets:                 targets,
		NormalizationSpec:       spec,
		ObservationFeatureNames: spec.ObservationFeatureNames(),
		ActionFeatureNames:      spec.ActionFeatureNames(),
		Samples:                 len(examples),
		ObsDim:                  obsDim,
		ActionDim:               actionDim,
		InputDim:                inputDim,
		TargetMin:               targetMin,
		TargetMax:               targetMax,
		ContinuedTransitions:    summary.ContinuedTransitions,
		TerminalTransitions:     summary.TerminalTransitions,
		UnlinkedTransitions:     summary.UnlinkedTransitions,
	}, nil
}

func loadTrainingTransitionRecords(ctx context.Context, source InputSourceConfig) ([]rl.TrainingTransitionRecord, error) {
	switch source.Format {
	case InputFormatJSONL:
		return loadTrainingTransitionRecordsFromJSONL(source)
	case InputFormatClickHouse:
		return loadTrainingTransitionRecordsFromClickHouse(ctx, source)
	default:
		return nil, fmt.Errorf("unsupported input format %q", source.Format)
	}
}

func loadTrainingTransitionRecordsFromJSONL(source InputSourceConfig) ([]rl.TrainingTransitionRecord, error) {
	var (
		reader *os.File
		err    error
	)
	if source.Path == "-" {
		reader = os.Stdin
	} else {
		reader, err = os.Open(source.Path)
		if err != nil {
			return nil, fmt.Errorf("open training transition jsonl %q: %w", source.Path, err)
		}
		defer func() {
			_ = reader.Close()
		}()
	}

	records := make([]rl.TrainingTransitionRecord, 0)
	_, err = rl.StreamTrainingTransitionsJSONL(reader, func(record rl.TrainingTransitionRecord) error {
		if !trainingTransitionMatchesQuery(record, source.Query) {
			return nil
		}
		records = append(records, record)
		if source.Query.Limit > 0 && len(records) >= source.Query.Limit {
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return nil, err
	}
	return records, nil
}

func loadTrainingTransitionRecordsFromClickHouse(ctx context.Context, source InputSourceConfig) ([]rl.TrainingTransitionRecord, error) {
	clickhouseConfig := rl.LoadClickHouseConfigFromEnv(os.Getenv)
	reader, err := rl.NewClickHouseTransitionReader(ctx, clickhouseConfig)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()

	records := make([]rl.TrainingTransitionRecord, 0)
	_, err = reader.StreamTransitions(ctx, source.Query, func(record rl.TrainingTransitionRecord) error {
		records = append(records, record)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

func trainingTransitionMatchesQuery(record rl.TrainingTransitionRecord, query rl.TrainingTransitionQuery) bool {
	if query.Scenario != "" && record.Scenario != query.Scenario {
		return false
	}
	if query.Outcome != "" && record.Outcome != query.Outcome {
		return false
	}
	if query.EpisodeIDMin > 0 && record.EpisodeID < query.EpisodeIDMin {
		return false
	}
	if query.EpisodeIDMax > 0 && record.EpisodeID > query.EpisodeIDMax {
		return false
	}
	return true
}

func clampFloat32(value, minValue, maxValue float32) float32 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func minFloat32(left, right float32) float32 {
	if left < right {
		return left
	}
	return right
}

func maxFloat32(left, right float32) float32 {
	if left > right {
		return left
	}
	return right
}
