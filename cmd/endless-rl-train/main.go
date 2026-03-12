package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/unng-lab/endless/pkg/rl"
)

func main() {
	config := rl.DuelRunConfig{}
	mode := "collect"
	policyName := rl.PolicyLeadAndStrafe
	exportFormat := string(rl.TransitionExportFormatJSONL)
	exportOutputPath := "-"
	exportScenario := ""
	exportOutcome := ""
	exportLimit := 0
	exportEpisodeIDMin := uint64(0)
	exportEpisodeIDMax := uint64(0)
	sequenceLimitEpisodes := 0
	sequenceMaxSteps := 0
	tensorBatchSize := 64
	policySuite := strings.Join([]string{rl.PolicyLeadAndStrafe, rl.PolicyRandom}, ",")
	compareBaselinePolicy := ""
	linearQStubDefaults := rl.DefaultLinearQStubConfig()
	trainEpochs := linearQStubDefaults.Epochs
	trainLearningRate := float64(linearQStubDefaults.LearningRate)
	trainDiscount := float64(linearQStubDefaults.Discount)
	flag.StringVar(&mode, "mode", "collect", "launcher mode: collect, evaluate, compare, export, export-sequences, inspect-batches or train-stub")
	flag.StringVar(&policyName, "policy", rl.PolicyLeadAndStrafe, "shooter policy: lead_strafe or random")
	flag.StringVar(&policySuite, "policy-suite", policySuite, "comma-separated policy list for compare mode")
	flag.StringVar(&compareBaselinePolicy, "compare-baseline-policy", "", "optional baseline policy inside compare mode; defaults to the first policy from -policy-suite")
	flag.IntVar(&config.Episodes, "episodes", 100, "number of duel episodes to generate")
	flag.Int64Var(&config.MaxTicksPerEpisode, "max-ticks", 600, "maximum simulation ticks per episode")
	flag.Int64Var(&config.Seed, "seed", time.Now().UnixNano(), "seed for deterministic duel generation")
	flag.IntVar(&config.WorldColumns, "world-columns", 64, "world column count for duel episodes")
	flag.IntVar(&config.WorldRows, "world-rows", 64, "world row count for duel episodes")
	flag.Float64Var(&config.TileSize, "tile-size", 16, "world tile size for duel episodes")
	flag.StringVar(&config.Scenario, "scenario", rl.DuelScenarioOpen, "duel scenario: duel_open or duel_with_cover")
	flag.StringVar(&exportFormat, "export-format", string(rl.TransitionExportFormatJSONL), "transition export format: jsonl or json")
	flag.StringVar(&exportOutputPath, "export-output", "-", "transition export destination path or - for stdout")
	flag.StringVar(&exportScenario, "export-scenario", "", "optional scenario filter for transition export")
	flag.StringVar(&exportOutcome, "export-outcome", "", "optional outcome filter for transition export")
	flag.IntVar(&exportLimit, "export-limit", 0, "optional transition export row limit; 0 exports all matched rows")
	flag.Uint64Var(&exportEpisodeIDMin, "export-episode-id-min", 0, "optional inclusive lower episode_id bound for transition export")
	flag.Uint64Var(&exportEpisodeIDMax, "export-episode-id-max", 0, "optional inclusive upper episode_id bound for transition export")
	flag.IntVar(&sequenceLimitEpisodes, "sequence-limit-episodes", 0, "optional episode limit for export-sequences mode; 0 exports all matched episodes")
	flag.IntVar(&sequenceMaxSteps, "sequence-max-steps", 0, "optional max transition count per exported sequence window; 0 keeps full episodes")
	flag.IntVar(&tensorBatchSize, "batch-size", 64, "tensor batch size for inspect-batches and train-stub modes")
	flag.IntVar(&trainEpochs, "train-epochs", linearQStubDefaults.Epochs, "number of offline learner epochs for train-stub mode")
	flag.Float64Var(&trainLearningRate, "train-learning-rate", float64(linearQStubDefaults.LearningRate), "learning rate for train-stub mode")
	flag.Float64Var(&trainDiscount, "train-discount", float64(linearQStubDefaults.Discount), "discount factor for train-stub mode")
	flag.Parse()

	ctx := context.Background()
	if mode == "export" {
		clickhouseConfig := rl.LoadClickHouseConfigFromEnv(os.Getenv)
		reader, err := rl.NewClickHouseTransitionReader(ctx, clickhouseConfig)
		if err != nil {
			log.Fatalf("create clickhouse transition reader: %v", err)
		}
		defer func() {
			_ = reader.Close()
		}()

		outputWriter, closeOutput, err := openTransitionExportDestination(exportOutputPath)
		if err != nil {
			log.Fatalf("open transition export destination %q: %v", exportOutputPath, err)
		}
		defer closeOutput()

		summary, err := reader.ExportTransitions(
			ctx,
			outputWriter,
			rl.TrainingTransitionQuery{
				Scenario:     exportScenario,
				Outcome:      exportOutcome,
				EpisodeIDMin: exportEpisodeIDMin,
				EpisodeIDMax: exportEpisodeIDMax,
				Limit:        exportLimit,
			},
			rl.TransitionExportFormat(exportFormat),
		)
		if err != nil {
			log.Fatalf("export transitions: %v", err)
		}

		log.Printf(
			"[rl-export] rows=%d format=%s output=%s scenario=%q outcome=%q episode_id_min=%d episode_id_max=%d limit=%d",
			summary.RowsExported,
			summary.Format,
			exportOutputPath,
			exportScenario,
			exportOutcome,
			exportEpisodeIDMin,
			exportEpisodeIDMax,
			exportLimit,
		)
		return
	}
	if mode == "export-sequences" {
		clickhouseConfig := rl.LoadClickHouseConfigFromEnv(os.Getenv)
		reader, err := rl.NewClickHouseTransitionReader(ctx, clickhouseConfig)
		if err != nil {
			log.Fatalf("create clickhouse transition reader: %v", err)
		}
		defer func() {
			_ = reader.Close()
		}()

		outputWriter, closeOutput, err := openTransitionExportDestination(exportOutputPath)
		if err != nil {
			log.Fatalf("open transition export destination %q: %v", exportOutputPath, err)
		}
		defer closeOutput()

		summary, err := reader.ExportTransitionSequences(
			ctx,
			outputWriter,
			rl.TrainingTransitionSequenceQuery{
				TransitionQuery: rl.TrainingTransitionQuery{
					Scenario:     exportScenario,
					Outcome:      exportOutcome,
					EpisodeIDMin: exportEpisodeIDMin,
					EpisodeIDMax: exportEpisodeIDMax,
					Limit:        exportLimit,
				},
				EpisodeLimit:      sequenceLimitEpisodes,
				MaxSequenceLength: sequenceMaxSteps,
			},
			rl.TransitionExportFormat(exportFormat),
		)
		if err != nil {
			log.Fatalf("export transition sequences: %v", err)
		}

		log.Printf(
			"[rl-export-sequences] episodes=%d sequences=%d rows=%d format=%s output=%s scenario=%q outcome=%q episode_id_min=%d episode_id_max=%d row_limit=%d episode_limit=%d max_steps=%d",
			summary.EpisodesExported,
			summary.SequencesExported,
			summary.RowsExported,
			summary.Format,
			exportOutputPath,
			exportScenario,
			exportOutcome,
			exportEpisodeIDMin,
			exportEpisodeIDMax,
			exportLimit,
			sequenceLimitEpisodes,
			sequenceMaxSteps,
		)
		return
	}
	if mode == "inspect-batches" {
		clickhouseConfig := rl.LoadClickHouseConfigFromEnv(os.Getenv)
		reader, err := rl.NewClickHouseTransitionReader(ctx, clickhouseConfig)
		if err != nil {
			log.Fatalf("create clickhouse transition reader: %v", err)
		}
		defer func() {
			_ = reader.Close()
		}()

		spec := rl.DefaultTransitionNormalizationSpec()
		batchBuilder, err := rl.NewTransitionBatchBuilder(spec, tensorBatchSize)
		if err != nil {
			log.Fatalf("create transition batch builder: %v", err)
		}

		inspection := rl.TransitionTensorInspection{}
		_, err = reader.StreamTransitions(
			ctx,
			rl.TrainingTransitionQuery{
				Scenario:     exportScenario,
				Outcome:      exportOutcome,
				EpisodeIDMin: exportEpisodeIDMin,
				EpisodeIDMax: exportEpisodeIDMax,
				Limit:        exportLimit,
			},
			func(record rl.TrainingTransitionRecord) error {
				transition, err := rl.VectorizeTransition(record, spec)
				if err != nil {
					return err
				}
				inspection.ObserveTransition(transition)

				completedBatch, err := batchBuilder.AppendTransition(transition)
				if err != nil {
					return err
				}
				inspection.ObserveCompletedBatch(completedBatch)
				return nil
			},
		)
		if err != nil {
			log.Fatalf("inspect transition batches: %v", err)
		}
		inspection.ObserveTailBatch(batchBuilder.Flush())

		log.Printf(
			"[rl-inspect] rows=%d completed_batches=%d tail_batch=%d obs_dim=%d action_dim=%d action_accepted=%d done=%d reward_range=[%.3f,%.3f] obs_range=[%.3f,%.3f] action_range=[%.3f,%.3f] next_obs_range=[%.3f,%.3f]",
			inspection.Rows,
			inspection.CompletedBatches,
			inspection.TailBatchSize,
			inspection.ObsDim,
			inspection.ActionDim,
			inspection.ActionAcceptedCount,
			inspection.DoneCount,
			inspection.RewardMin,
			inspection.RewardMax,
			inspection.ObsMin,
			inspection.ObsMax,
			inspection.ActionMin,
			inspection.ActionMax,
			inspection.NextObsMin,
			inspection.NextObsMax,
		)
		return
	}
	if mode == "train-stub" {
		clickhouseConfig := rl.LoadClickHouseConfigFromEnv(os.Getenv)
		reader, err := rl.NewClickHouseTransitionReader(ctx, clickhouseConfig)
		if err != nil {
			log.Fatalf("create clickhouse transition reader: %v", err)
		}
		defer func() {
			_ = reader.Close()
		}()

		spec := rl.DefaultTransitionNormalizationSpec()
		dataset, err := rl.LoadVectorizedTransitionDataset(
			ctx,
			reader,
			rl.TrainingTransitionQuery{
				Scenario:     exportScenario,
				Outcome:      exportOutcome,
				EpisodeIDMin: exportEpisodeIDMin,
				EpisodeIDMax: exportEpisodeIDMax,
				Limit:        exportLimit,
			},
			spec,
		)
		if err != nil {
			log.Fatalf("load vectorized transition dataset: %v", err)
		}

		model, summary, err := rl.TrainLinearQStub(ctx, dataset.Transitions, rl.LinearQStubConfig{
			Epochs:       trainEpochs,
			BatchSize:    tensorBatchSize,
			LearningRate: float32(trainLearningRate),
			Discount:     float32(trainDiscount),
		})
		if err != nil {
			log.Fatalf("train linear q stub: %v", err)
		}

		for _, epoch := range summary.EpochSummaries {
			log.Printf(
				"[rl-train-stub-epoch] epoch=%d loss=%.6f td_mae=%.6f avg_prediction=%.6f avg_target=%.6f",
				epoch.Epoch,
				epoch.AverageLoss,
				epoch.MeanAbsoluteTDError,
				epoch.AveragePrediction,
				epoch.AverageTarget,
			)
		}

		log.Printf(
			"[rl-train-stub] rows=%d linked_next_actions=%d terminal=%d unlinked=%d obs_dim=%d action_dim=%d input_dim=%d weights=%d initial_loss=%.6f final_loss=%.6f final_td_mae=%.6f prediction_range=[%.6f,%.6f] target_range=[%.6f,%.6f] batch_size=%d epochs=%d lr=%.4f discount=%.4f",
			len(dataset.Transitions),
			summary.LinkedNextActions,
			summary.TerminalTransitions,
			summary.UnlinkedTransitions,
			summary.ObsDim,
			summary.ActionDim,
			summary.InputDim,
			len(model.Weights),
			summary.InitialAverageLoss,
			summary.FinalAverageLoss,
			summary.FinalMeanAbsoluteTD,
			summary.PredictionMin,
			summary.PredictionMax,
			summary.TargetMin,
			summary.TargetMax,
			summary.BatchSize,
			summary.Epochs,
			trainLearningRate,
			trainDiscount,
		)
		return
	}
	if mode == "compare" {
		suiteSummary, err := rl.RunDuelPolicyComparisonSuite(
			ctx,
			config,
			rl.ParsePolicyNameList(policySuite),
			compareBaselinePolicy,
		)
		if err != nil {
			log.Fatalf("run duel policy comparison suite: %v", err)
		}

		for _, result := range suiteSummary.Results {
			log.Printf(
				"[rl-compare] suite_seed=%d episodes=%d policy=%s target_kills=%d shooter_deaths=%d timeouts=%d shots=%d hits=%d expired=%d avg_reward=%.3f avg_ticks=%.1f",
				suiteSummary.SuiteSeed,
				result.Summary.EpisodesGenerated,
				result.PolicyName,
				result.Summary.TargetKills,
				result.Summary.ShooterDeaths,
				result.Summary.Timeouts,
				result.Summary.ShotsFired,
				result.Summary.ProjectileHits,
				result.Summary.ProjectileExpired,
				result.Summary.AverageReward,
				result.Summary.AverageTicks,
			)
		}
		for _, comparison := range suiteSummary.Comparisons {
			log.Printf(
				"[rl-compare-delta] baseline=%s policy=%s avg_reward_delta=%.3f avg_ticks_delta=%.1f total_reward_delta=%.3f target_kills_delta=%d shooter_deaths_delta=%d timeouts_delta=%d shots_delta=%d hits_delta=%d expired_delta=%d",
				comparison.BaselinePolicyName,
				comparison.PolicyName,
				comparison.AverageRewardDelta,
				comparison.AverageTicksDelta,
				comparison.TotalRewardDelta,
				comparison.TargetKillsDelta,
				comparison.ShooterDeathsDelta,
				comparison.TimeoutsDelta,
				comparison.ShotsFiredDelta,
				comparison.ProjectileHitsDelta,
				comparison.ProjectileExpiredDelta,
			)
		}
		return
	}

	policy, err := rl.NewPolicyByName(policyName, config.Seed)
	if err != nil {
		log.Fatalf("create policy %q: %v", policyName, err)
	}

	if mode == "evaluate" {
		summary, err := rl.RunDuelEvaluation(ctx, config, policy)
		if err != nil {
			log.Fatalf("run duel evaluation: %v", err)
		}

		log.Printf(
			"[rl-eval] episodes=%d target_kills=%d shooter_deaths=%d timeouts=%d shots=%d hits=%d expired=%d avg_reward=%.3f avg_ticks=%.1f",
			summary.EpisodesGenerated,
			summary.TargetKills,
			summary.ShooterDeaths,
			summary.Timeouts,
			summary.ShotsFired,
			summary.ProjectileHits,
			summary.ProjectileExpired,
			summary.AverageReward,
			summary.AverageTicks,
		)
		return
	}
	if mode != "collect" {
		log.Fatalf("unsupported mode %q; use collect, evaluate, compare, export, export-sequences, inspect-batches or train-stub", mode)
	}

	clickhouseConfig := rl.LoadClickHouseConfigFromEnv(os.Getenv)
	recorder, err := rl.NewClickHouseRecorder(ctx, clickhouseConfig)
	if err != nil {
		log.Fatalf("create clickhouse recorder: %v", err)
	}

	if err := rl.RunDuelCollectionWithPolicy(ctx, config, recorder, policy); err != nil {
		log.Fatalf("run duel collection: %v", err)
	}
}

// openTransitionExportDestination keeps the export path handling explicit so the launcher may
// stream directly to stdout for pipes or to a file for trainer-side batch jobs.
func openTransitionExportDestination(path string) (*os.File, func(), error) {
	if path == "" || path == "-" {
		return os.Stdout, func() {}, nil
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return file, func() {
		_ = file.Close()
	}, nil
}
