package rl

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/unng-lab/endless/pkg/unit"
)

// DuelEvaluationSummary keeps a compact aggregate over multiple deterministic duel episodes so
// launchers may compare policies without persisting the full transition dataset.
type DuelEvaluationSummary struct {
	EpisodesGenerated int
	TargetKills       int
	ShooterDeaths     int
	Timeouts          int
	ShotsFired        int
	ProjectileHits    int
	ProjectileExpired int
	TotalReward       float64
	AverageReward     float64
	AverageTicks      float64
}

// RunDuelEvaluation executes the same environment loop as collection mode but aggregates only
// high-level metrics that are cheap to print in CLI tools and regression checks.
func RunDuelEvaluation(ctx context.Context, config DuelRunConfig, policy Policy) (DuelEvaluationSummary, error) {
	config = normalizedDuelRunConfig(config)
	if policy == nil {
		policy = NewLeadAndStrafePolicy()
	}

	return runDuelEvaluationWithEpisodeSeeds(ctx, config, policy, generateDuelEvaluationEpisodeSeeds(config.Seed, config.Episodes))
}

// generateDuelEvaluationEpisodeSeeds freezes the episode suite derived from one root seed so
// multiple policies may be evaluated against the exact same duel instances for fair comparison.
func generateDuelEvaluationEpisodeSeeds(seed int64, episodes int) []int64 {
	if episodes <= 0 {
		return nil
	}

	sessionRNG := rand.New(rand.NewSource(seed))
	episodeSeeds := make([]int64, 0, episodes)
	for episodeIndex := 0; episodeIndex < episodes; episodeIndex++ {
		episodeSeeds = append(episodeSeeds, sessionRNG.Int63())
	}
	return episodeSeeds
}

// runDuelEvaluationWithEpisodeSeeds replays one explicit episode seed suite so callers can
// compare multiple policies against the same deterministic duel lineup without regenerating a
// different world sequence per policy.
func runDuelEvaluationWithEpisodeSeeds(ctx context.Context, config DuelRunConfig, policy Policy, episodeSeeds []int64) (DuelEvaluationSummary, error) {
	config = normalizedDuelRunConfig(config)
	if policy == nil {
		policy = NewLeadAndStrafePolicy()
	}

	summary := DuelEvaluationSummary{}
	totalTicks := uint64(0)
	for episodeIndex, episodeSeed := range episodeSeeds {
		select {
		case <-ctx.Done():
			return summary, ctx.Err()
		default:
		}

		episodeSummary, err := runOneDuelEvaluationEpisode(episodeSeed, config, policy)
		if err != nil {
			return summary, fmt.Errorf("run duel evaluation episode %d: %w", episodeIndex+1, err)
		}

		summary.EpisodesGenerated++
		summary.TargetKills += episodeSummary.TargetKills
		summary.ShooterDeaths += episodeSummary.ShooterDeaths
		summary.Timeouts += episodeSummary.Timeouts
		summary.ShotsFired += episodeSummary.ShotsFired
		summary.ProjectileHits += episodeSummary.ProjectileHits
		summary.ProjectileExpired += episodeSummary.ProjectileExpired
		summary.TotalReward += episodeSummary.TotalReward
		totalTicks += uint64(episodeSummary.Ticks)
	}

	if summary.EpisodesGenerated > 0 {
		summary.AverageReward = summary.TotalReward / float64(summary.EpisodesGenerated)
		summary.AverageTicks = float64(totalTicks) / float64(summary.EpisodesGenerated)
	}

	return summary, nil
}

type duelEpisodeEvaluationSummary struct {
	TargetKills       int
	ShooterDeaths     int
	Timeouts          int
	ShotsFired        int
	ProjectileHits    int
	ProjectileExpired int
	TotalReward       float64
	Ticks             uint32
}

func runOneDuelEvaluationEpisode(episodeSeed int64, config DuelRunConfig, policy Policy) (duelEpisodeEvaluationSummary, error) {
	environment := NewDuelEnvironment(config)
	defer environment.Close()

	before, err := environment.Reset(episodeSeed)
	if err != nil {
		return duelEpisodeEvaluationSummary{}, err
	}

	summary := duelEpisodeEvaluationSummary{}
	for tick := int64(1); tick <= config.MaxTicksPerEpisode; tick++ {
		action := policy.ChooseAction(before)
		// Evaluation cares about resulting gameplay metrics. Invalid commands still manifest
		// through missing progress and failed order reports inside the environment itself.
		_, _ = environment.ApplyAction(action)

		stepResult, err := environment.Step()
		if err != nil {
			return duelEpisodeEvaluationSummary{}, err
		}

		summary.Ticks = uint32(stepResult.After.Snapshot.Tick)
		summary.TotalReward += float64(stepResult.Reward)
		accumulateCombatMetrics(&summary, stepResult.CombatEvents)
		before = stepResult.After
		if !stepResult.Done {
			continue
		}

		switch stepResult.Outcome {
		case "target_killed":
			summary.TargetKills++
		case "shooter_killed":
			summary.ShooterDeaths++
		case "timeout":
			summary.Timeouts++
		}
		break
	}

	return summary, nil
}

func accumulateCombatMetrics(summary *duelEpisodeEvaluationSummary, events []unit.CombatEvent) {
	if summary == nil {
		return
	}

	for _, event := range events {
		switch event.Type {
		case unit.CombatEventProjectileSpawned:
			summary.ShotsFired++
		case unit.CombatEventProjectileHit:
			summary.ProjectileHits++
		case unit.CombatEventProjectileExpired:
			summary.ProjectileExpired++
		}
	}
}
