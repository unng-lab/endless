package rl

import (
	"context"
	"reflect"
	"testing"
)

func TestParsePolicyNameListNormalizesAndDeduplicates(t *testing.T) {
	names := ParsePolicyNameList(" lead_strafe, RANDOM ,lead_strafe,,random ")
	expected := []string{PolicyLeadAndStrafe, PolicyRandom}
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("ParsePolicyNameList() = %#v, want %#v", names, expected)
	}
}

func TestRunDuelPolicyComparisonSuiteIsDeterministicForFixedSeed(t *testing.T) {
	config := DuelRunConfig{
		Episodes:           2,
		MaxTicksPerEpisode: 90,
		Seed:               17,
		WorldColumns:       64,
		WorldRows:          64,
		TileSize:           16,
		Scenario:           DuelScenarioWithCover,
	}

	first, err := RunDuelPolicyComparisonSuite(
		context.Background(),
		config,
		[]string{PolicyLeadAndStrafe, PolicyRandom},
		PolicyLeadAndStrafe,
	)
	if err != nil {
		t.Fatalf("first RunDuelPolicyComparisonSuite() error = %v", err)
	}
	second, err := RunDuelPolicyComparisonSuite(
		context.Background(),
		config,
		[]string{PolicyLeadAndStrafe, PolicyRandom},
		PolicyLeadAndStrafe,
	)
	if err != nil {
		t.Fatalf("second RunDuelPolicyComparisonSuite() error = %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("RunDuelPolicyComparisonSuite() is not deterministic:\nfirst=%#v\nsecond=%#v", first, second)
	}
	if got, want := len(first.EpisodeSeeds), config.Episodes; got != want {
		t.Fatalf("len(EpisodeSeeds) = %d, want %d", got, want)
	}
	if got, want := len(first.Results), 2; got != want {
		t.Fatalf("len(Results) = %d, want %d", got, want)
	}
	if got, want := len(first.Comparisons), 2; got != want {
		t.Fatalf("len(Comparisons) = %d, want %d", got, want)
	}
}

func TestRunDuelPolicyComparisonSuiteUsesRequestedBaseline(t *testing.T) {
	suiteSummary, err := RunDuelPolicyComparisonSuite(
		context.Background(),
		DuelRunConfig{
			Episodes:           2,
			MaxTicksPerEpisode: 90,
			Seed:               23,
			WorldColumns:       64,
			WorldRows:          64,
			TileSize:           16,
		},
		[]string{PolicyLeadAndStrafe, PolicyRandom},
		PolicyRandom,
	)
	if err != nil {
		t.Fatalf("RunDuelPolicyComparisonSuite() error = %v", err)
	}

	if got, want := suiteSummary.BaselinePolicyName, PolicyRandom; got != want {
		t.Fatalf("BaselinePolicyName = %q, want %q", got, want)
	}

	foundZeroBaselineDelta := false
	for _, comparison := range suiteSummary.Comparisons {
		if comparison.PolicyName != PolicyRandom {
			continue
		}
		foundZeroBaselineDelta = true
		if comparison.AverageRewardDelta != 0 || comparison.AverageTicksDelta != 0 || comparison.TotalRewardDelta != 0 {
			t.Fatalf("baseline delta = %#v, want zero deltas", comparison)
		}
	}
	if !foundZeroBaselineDelta {
		t.Fatal("expected comparison entry for requested baseline policy")
	}
}
