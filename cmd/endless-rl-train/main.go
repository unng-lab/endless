package main

import (
	"context"
	"flag"
	"log"
	"os"
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
	tensorBatchSize := 64
	flag.StringVar(&mode, "mode", "collect", "launcher mode: collect, evaluate, export or inspect-batches")
	flag.StringVar(&policyName, "policy", rl.PolicyLeadAndStrafe, "shooter policy: lead_strafe or random")
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
	flag.IntVar(&tensorBatchSize, "batch-size", 64, "tensor batch size for inspect-batches mode")
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
		log.Fatalf("unsupported mode %q; use collect, evaluate, export or inspect-batches", mode)
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
