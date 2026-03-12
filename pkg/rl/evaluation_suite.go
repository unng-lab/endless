package rl

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
)

// DuelPolicyEvaluationResult keeps one named policy summary inside a fixed-seed comparison
// suite so CLI tools can print per-policy metrics and later compare them against a baseline.
type DuelPolicyEvaluationResult struct {
	PolicyName string
	Summary    DuelEvaluationSummary
}

// DuelPolicyComparison keeps the most operator-relevant deltas versus one chosen baseline.
// The fields stay intentionally compact so CLI logs can highlight reward and outcome shifts
// without dumping full per-episode traces.
type DuelPolicyComparison struct {
	BaselinePolicyName     string
	PolicyName             string
	TargetKillsDelta       int
	ShooterDeathsDelta     int
	TimeoutsDelta          int
	ShotsFiredDelta        int
	ProjectileHitsDelta    int
	ProjectileExpiredDelta int
	TotalRewardDelta       float64
	AverageRewardDelta     float64
	AverageTicksDelta      float64
}

// DuelPolicyComparisonSuiteSummary reports one fully deterministic multi-policy evaluation
// suite. Every policy sees the same generated episode seed lineup so training regressions can
// be measured fairly from the headless launcher.
type DuelPolicyComparisonSuiteSummary struct {
	SuiteSeed          int64
	EpisodeSeeds       []int64
	BaselinePolicyName string
	Results            []DuelPolicyEvaluationResult
	Comparisons        []DuelPolicyComparison
}

// ParsePolicyNameList normalizes one comma-separated CLI policy list into a deterministic
// unique order so suite evaluation always compares the same policy set for the same input.
func ParsePolicyNameList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	seen := map[string]struct{}{}
	names := make([]string, 0, 4)
	for _, part := range strings.Split(value, ",") {
		name := normalizedPolicyName(part)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

// RunDuelPolicyComparisonSuite evaluates every requested policy against one shared fixed-seed
// episode suite. The first listed policy becomes the default baseline unless the caller names
// another supported baseline explicitly.
func RunDuelPolicyComparisonSuite(ctx context.Context, config DuelRunConfig, policyNames []string, baselinePolicyName string) (DuelPolicyComparisonSuiteSummary, error) {
	config = normalizedDuelRunConfig(config)
	policyNames = normalizedPolicyNameList(policyNames)
	if len(policyNames) == 0 {
		policyNames = []string{PolicyLeadAndStrafe, PolicyRandom}
	}

	episodeSeeds := generateDuelEvaluationEpisodeSeeds(config.Seed, config.Episodes)
	results := make([]DuelPolicyEvaluationResult, 0, len(policyNames))
	for policyIndex, policyName := range policyNames {
		select {
		case <-ctx.Done():
			return DuelPolicyComparisonSuiteSummary{}, ctx.Err()
		default:
		}

		policy, err := NewPolicyByName(policyName, deterministicPolicySeed(config.Seed, policyName, policyIndex))
		if err != nil {
			return DuelPolicyComparisonSuiteSummary{}, fmt.Errorf("create policy %q: %w", policyName, err)
		}
		summary, err := runDuelEvaluationWithEpisodeSeeds(ctx, config, policy, episodeSeeds)
		if err != nil {
			return DuelPolicyComparisonSuiteSummary{}, fmt.Errorf("evaluate policy %q: %w", policyName, err)
		}
		results = append(results, DuelPolicyEvaluationResult{
			PolicyName: policyName,
			Summary:    summary,
		})
	}

	baselinePolicyName = normalizedPolicyName(baselinePolicyName)
	if baselinePolicyName == "" {
		baselinePolicyName = results[0].PolicyName
	}

	baselineIndex := -1
	for index, result := range results {
		if result.PolicyName == baselinePolicyName {
			baselineIndex = index
			break
		}
	}
	if baselineIndex < 0 {
		return DuelPolicyComparisonSuiteSummary{}, fmt.Errorf("baseline policy %q is not present in suite %q", baselinePolicyName, strings.Join(policyNames, ","))
	}

	suiteSummary := DuelPolicyComparisonSuiteSummary{
		SuiteSeed:          config.Seed,
		EpisodeSeeds:       append([]int64(nil), episodeSeeds...),
		BaselinePolicyName: baselinePolicyName,
		Results:            results,
		Comparisons:        make([]DuelPolicyComparison, 0, len(results)),
	}

	baselineSummary := results[baselineIndex].Summary
	for _, result := range results {
		suiteSummary.Comparisons = append(suiteSummary.Comparisons, DuelPolicyComparison{
			BaselinePolicyName:     baselinePolicyName,
			PolicyName:             result.PolicyName,
			TargetKillsDelta:       result.Summary.TargetKills - baselineSummary.TargetKills,
			ShooterDeathsDelta:     result.Summary.ShooterDeaths - baselineSummary.ShooterDeaths,
			TimeoutsDelta:          result.Summary.Timeouts - baselineSummary.Timeouts,
			ShotsFiredDelta:        result.Summary.ShotsFired - baselineSummary.ShotsFired,
			ProjectileHitsDelta:    result.Summary.ProjectileHits - baselineSummary.ProjectileHits,
			ProjectileExpiredDelta: result.Summary.ProjectileExpired - baselineSummary.ProjectileExpired,
			TotalRewardDelta:       result.Summary.TotalReward - baselineSummary.TotalReward,
			AverageRewardDelta:     result.Summary.AverageReward - baselineSummary.AverageReward,
			AverageTicksDelta:      result.Summary.AverageTicks - baselineSummary.AverageTicks,
		})
	}

	return suiteSummary, nil
}

func normalizedPolicyNameList(policyNames []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(policyNames))
	for _, candidate := range policyNames {
		name := normalizedPolicyName(candidate)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	return normalized
}

// deterministicPolicySeed derives a stable constructor seed for stochastic policies without
// relying on slice position alone. This keeps compare mode reproducible even if the policy set
// grows or the caller changes the baseline ordering.
func deterministicPolicySeed(suiteSeed int64, policyName string, policyIndex int) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(policyName))
	nameHash := int64(hasher.Sum64())
	return suiteSeed + nameHash + int64(policyIndex+1)*7919
}
