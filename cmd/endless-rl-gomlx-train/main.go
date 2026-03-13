package main

import (
	"context"
	"flag"
	"log"
	"strconv"
	"strings"

	"github.com/unng-lab/endless/pkg/rl"
	"github.com/unng-lab/endless/pkg/rl/gomlxtrain"
)

func main() {
	defaults := gomlxtrain.DefaultConfig()

	inputFormat := string(defaults.Source.Format)
	inputPath := ""
	backend := ""
	hiddenDims := joinInts(defaults.HiddenDims)
	discount := float64(defaults.Discount)
	targetAbsMax := float64(defaults.TargetAbsMax)

	cfg := gomlxtrain.Config{
		Source: gomlxtrain.InputSourceConfig{
			Query: rl.TrainingTransitionQuery{},
		},
		CheckpointDir:  defaults.CheckpointDir,
		CheckpointKeep: defaults.CheckpointKeep,
		BatchSize:      defaults.BatchSize,
		Epochs:         defaults.Epochs,
		LearningRate:   defaults.LearningRate,
		Discount:       defaults.Discount,
		TargetAbsMax:   defaults.TargetAbsMax,
		HiddenDims:     defaults.HiddenDims,
	}

	flag.StringVar(&inputFormat, "input-format", string(defaults.Source.Format), "trainer input format: jsonl or clickhouse")
	flag.StringVar(&inputPath, "input-path", "", "path to trainer input JSONL file or - for stdin; ignored for clickhouse")
	flag.StringVar(&backend, "backend", "", "optional GOMLX_BACKEND override, e.g. xla:cuda or xla:cpu")
	flag.StringVar(&cfg.CheckpointDir, "checkpoint-dir", defaults.CheckpointDir, "directory where GoMLX checkpoints and manifest are written")
	flag.IntVar(&cfg.CheckpointKeep, "checkpoint-keep", defaults.CheckpointKeep, "how many checkpoint snapshots to keep inside checkpoint-dir")
	flag.IntVar(&cfg.BatchSize, "batch-size", defaults.BatchSize, "training batch size for the external GoMLX trainer")
	flag.IntVar(&cfg.Epochs, "train-epochs", defaults.Epochs, "number of training epochs for the external GoMLX trainer")
	flag.Float64Var(&cfg.LearningRate, "train-learning-rate", defaults.LearningRate, "learning rate for the external GoMLX trainer")
	flag.Float64Var(&discount, "train-discount", float64(defaults.Discount), "discount factor used to derive return-to-go targets")
	flag.Float64Var(&targetAbsMax, "target-abs-max", float64(defaults.TargetAbsMax), "optional absolute clip for discounted return targets; 0 disables clipping")
	flag.Int64Var(&cfg.Seed, "seed", 0, "optional deterministic trainer seed for dataset shuffling and parameter initialization")
	flag.StringVar(&hiddenDims, "hidden-dims", hiddenDims, "comma-separated hidden layer widths for the critic MLP, e.g. 256,128")
	flag.StringVar(&cfg.Source.Query.Scenario, "export-scenario", "", "optional scenario filter for trainer input")
	flag.StringVar(&cfg.Source.Query.Outcome, "export-outcome", "", "optional outcome filter for trainer input")
	flag.Uint64Var(&cfg.Source.Query.EpisodeIDMin, "export-episode-id-min", 0, "optional inclusive lower episode_id bound for trainer input")
	flag.Uint64Var(&cfg.Source.Query.EpisodeIDMax, "export-episode-id-max", 0, "optional inclusive upper episode_id bound for trainer input")
	flag.IntVar(&cfg.Source.Query.Limit, "export-limit", 0, "optional row limit for trainer input")
	flag.Parse()

	parsedHiddenDims, err := parseHiddenDims(hiddenDims)
	if err != nil {
		log.Fatalf("parse -hidden-dims: %v", err)
	}

	cfg.HiddenDims = parsedHiddenDims
	cfg.Backend = strings.TrimSpace(backend)
	cfg.Discount = float32(discount)
	cfg.TargetAbsMax = float32(targetAbsMax)
	cfg.Source.Format = gomlxtrain.InputFormat(strings.ToLower(strings.TrimSpace(inputFormat)))
	cfg.Source.Path = strings.TrimSpace(inputPath)

	result, err := gomlxtrain.TrainCritic(context.Background(), cfg)
	if err != nil {
		log.Fatalf("train GoMLX critic: %v", err)
	}

	log.Printf(
		"[rl-gomlx-train] backend=%s samples=%d obs_dim=%d action_dim=%d input_dim=%d continued=%d terminal=%d unlinked=%d target_range=[%.4f,%.4f] checkpoint_dir=%s manifest=%s start_step=%d end_step=%d",
		result.BackendName,
		result.Samples,
		result.ObsDim,
		result.ActionDim,
		result.InputDim,
		result.ContinuedTransitions,
		result.TerminalTransitions,
		result.UnlinkedTransitions,
		result.TargetMin,
		result.TargetMax,
		result.CheckpointDir,
		result.ManifestPath,
		result.StartStep,
		result.EndStep,
	)
	log.Printf("[rl-gomlx-train-backend] %s", result.BackendDescription)
}

func parseHiddenDims(value string) ([]int, error) {
	parts := strings.Split(strings.TrimSpace(value), ",")
	hiddenDims := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		hiddenDim, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		if hiddenDim <= 0 {
			return nil, strconv.ErrSyntax
		}
		hiddenDims = append(hiddenDims, hiddenDim)
	}
	if len(hiddenDims) == 0 {
		return nil, strconv.ErrSyntax
	}
	return hiddenDims, nil
}

func joinInts(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}
