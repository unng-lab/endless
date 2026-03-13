//go:build linux

package gomlxtrain

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/gomlx/gomlx/backends"
	_ "github.com/gomlx/gomlx/backends/default"
	gomlxgraph "github.com/gomlx/gomlx/pkg/core/graph"
	gomlxtensors "github.com/gomlx/gomlx/pkg/core/tensors"
	gomlxcontext "github.com/gomlx/gomlx/pkg/ml/context"
	"github.com/gomlx/gomlx/pkg/ml/context/checkpoints"
	"github.com/gomlx/gomlx/pkg/ml/datasets"
	"github.com/gomlx/gomlx/pkg/ml/layers"
	"github.com/gomlx/gomlx/pkg/ml/layers/activations"
	"github.com/gomlx/gomlx/pkg/ml/train"
	"github.com/gomlx/gomlx/pkg/ml/train/losses"
	"github.com/gomlx/gomlx/pkg/ml/train/optimizers"
	"github.com/gomlx/gomlx/ui/commandline"
)

const trainerManifestFileName = "gomlx_critic_manifest.json"
const trainerManifestVersion = 1

// TrainCritic trains an offline value critic on top of the stable `(obs||action)` tensor layout
// and persists the resulting GoMLX checkpoint plus a compact manifest that describes the contract.
func TrainCritic(ctx context.Context, cfg Config) (Result, error) {
	cfg, err := cfg.Normalized()
	if err != nil {
		return Result{}, err
	}

	prepared, err := PrepareDataset(ctx, cfg)
	if err != nil {
		return Result{}, err
	}

	if cfg.Backend != "" {
		if err := os.Setenv("GOMLX_BACKEND", cfg.Backend); err != nil {
			return Result{}, fmt.Errorf("set GOMLX_BACKEND=%q: %w", cfg.Backend, err)
		}
	}

	backend, err := backends.New()
	if err != nil {
		return Result{}, fmt.Errorf("create GoMLX backend: %w", err)
	}
	defer backend.Finalize()

	mlCtx := gomlxcontext.New()
	mlCtx.SetParam(optimizers.ParamOptimizer, "adam")
	mlCtx.SetParam(optimizers.ParamLearningRate, cfg.LearningRate)
	if cfg.Seed != 0 {
		mlCtx.SetParam(gomlxcontext.ParamInitialSeed, cfg.Seed)
	}
	if err := mlCtx.ResetRNGState(); err != nil {
		return Result{}, fmt.Errorf("reset GoMLX RNG state: %w", err)
	}

	checkpointHandler, err := checkpoints.Build(mlCtx).
		Dir(cfg.CheckpointDir).
		Keep(cfg.CheckpointKeep).
		Done()
	if err != nil {
		return Result{}, fmt.Errorf("create checkpoint handler: %w", err)
	}

	inputsTensor := gomlxtensors.FromFlatDataAndDimensions(prepared.Inputs, prepared.Samples, prepared.InputDim)
	targetsTensor := gomlxtensors.FromFlatDataAndDimensions(prepared.Targets, prepared.Samples, 1)
	defer inputsTensor.MustFinalizeAll()
	defer targetsTensor.MustFinalizeAll()

	dataset, err := datasets.InMemoryFromData(backend, "endless_gomlx_critic", []any{inputsTensor}, []any{targetsTensor})
	if err != nil {
		return Result{}, fmt.Errorf("create in-memory GoMLX dataset: %w", err)
	}
	defer dataset.FinalizeAll()

	batchPlan, err := resolveTrainingBatchPlan(prepared.Samples, cfg.BatchSize)
	if err != nil {
		return Result{}, err
	}
	cfg.BatchSize = batchPlan.BatchSize

	dataset.
		Shuffle().
		WithRand(newDatasetRand(cfg.Seed)).
		Infinite(true).
		BatchSize(batchPlan.BatchSize, batchPlan.DropIncompleteBatch)

	trainer := train.NewTrainer(
		backend,
		mlCtx,
		buildCriticModelGraph(cfg.HiddenDims),
		losses.MeanSquaredError,
		optimizers.FromContext(mlCtx),
		nil,
		nil,
	)

	stepsPerEpoch := batchPlan.StepsPerEpoch
	totalSteps := stepsPerEpoch * cfg.Epochs

	loop := train.NewLoop(trainer)
	commandline.AttachProgressBar(loop)
	train.EveryNSteps(loop, stepsPerEpoch, "gomlx checkpoint", 100, checkpointHandler.OnStepFn)

	startStep := optimizers.GetGlobalStep(mlCtx)
	if int(startStep) < totalSteps {
		if _, err := loop.RunSteps(dataset, totalSteps-int(startStep)); err != nil {
			return Result{}, fmt.Errorf("run GoMLX training loop: %w", err)
		}
	}
	if err := checkpointHandler.Save(); err != nil {
		return Result{}, fmt.Errorf("save final checkpoint: %w", err)
	}

	endStep := optimizers.GetGlobalStep(mlCtx)
	manifestPath := filepath.Join(cfg.CheckpointDir, trainerManifestFileName)
	manifest := Manifest{
		Version:                 trainerManifestVersion,
		TrainedAt:               time.Now().UTC(),
		Source:                  cfg.Source,
		NormalizationSpec:       prepared.NormalizationSpec,
		ObservationFeatureNames: append([]string(nil), prepared.ObservationFeatureNames...),
		ActionFeatureNames:      append([]string(nil), prepared.ActionFeatureNames...),
		ObsDim:                  prepared.ObsDim,
		ActionDim:               prepared.ActionDim,
		InputDim:                prepared.InputDim,
		HiddenDims:              append([]int(nil), cfg.HiddenDims...),
		Samples:                 prepared.Samples,
		ContinuedTransitions:    prepared.ContinuedTransitions,
		TerminalTransitions:     prepared.TerminalTransitions,
		UnlinkedTransitions:     prepared.UnlinkedTransitions,
		TargetMin:               prepared.TargetMin,
		TargetMax:               prepared.TargetMax,
		BatchSize:               cfg.BatchSize,
		Epochs:                  cfg.Epochs,
		LearningRate:            cfg.LearningRate,
		Discount:                cfg.Discount,
		TargetAbsMax:            cfg.TargetAbsMax,
		Seed:                    cfg.Seed,
		Backend:                 os.Getenv("GOMLX_BACKEND"),
		BackendName:             backend.Name(),
		BackendDescription:      backend.Description(),
		GlobalStep:              endStep,
		CheckpointDir:           cfg.CheckpointDir,
	}
	if err := writeTrainerManifest(manifestPath, manifest); err != nil {
		return Result{}, err
	}

	return Result{
		BackendName:          backend.Name(),
		BackendDescription:   backend.Description(),
		ManifestPath:         manifestPath,
		CheckpointDir:        cfg.CheckpointDir,
		Samples:              prepared.Samples,
		ObsDim:               prepared.ObsDim,
		ActionDim:            prepared.ActionDim,
		InputDim:             prepared.InputDim,
		ContinuedTransitions: prepared.ContinuedTransitions,
		TerminalTransitions:  prepared.TerminalTransitions,
		UnlinkedTransitions:  prepared.UnlinkedTransitions,
		TargetMin:            prepared.TargetMin,
		TargetMax:            prepared.TargetMax,
		StartStep:            startStep,
		EndStep:              endStep,
	}, nil
}

func buildCriticModelGraph(hiddenDims []int) func(ctx *gomlxcontext.Context, spec any, inputs []*gomlxgraph.Node) []*gomlxgraph.Node {
	hiddenDims = append([]int(nil), hiddenDims...)
	return func(ctx *gomlxcontext.Context, spec any, inputs []*gomlxgraph.Node) []*gomlxgraph.Node {
		_ = spec
		logits := inputs[0]
		for index, hiddenDim := range hiddenDims {
			logits = layers.Dense(ctx.In(fmt.Sprintf("hidden_%d", index)), logits, true, hiddenDim)
			logits = activations.Relu(logits)
		}
		logits = layers.Dense(ctx.In("value_head"), logits, true, 1)
		return []*gomlxgraph.Node{logits}
	}
}

// newDatasetRand keeps training deterministic when the operator supplies a seed, while still
// defaulting to non-repeating shuffle order when no seed was requested explicitly.
func newDatasetRand(seed int64) *rand.Rand {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return rand.New(rand.NewSource(seed))
}

func writeTrainerManifest(path string, manifest Manifest) error {
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal trainer manifest: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write trainer manifest %q: %w", path, err)
	}
	return nil
}
