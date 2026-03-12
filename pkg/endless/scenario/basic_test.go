package scenario

import (
	"math/rand"
	"testing"
)

func TestBasicScenarioRunnerTilesStayCenteredAsTwoByTwoGroup(t *testing.T) {
	scenario := &basicScenario{
		centerTileX: 50,
		centerTileY: 75,
		rng:         rand.New(rand.NewSource(1)),
	}

	got := scenario.runnerTiles()
	want := []basicTileAnchor{
		{tileX: 49, tileY: 74},
		{tileX: 50, tileY: 74},
		{tileX: 49, tileY: 75},
		{tileX: 50, tileY: 75},
	}

	if len(got) != len(want) {
		t.Fatalf("runnerTiles length = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("runnerTiles[%d] = %+v, want %+v", index, got[index], want[index])
		}
	}
}

func TestBasicScenarioStaticTilesReturnUniqueWallCandidates(t *testing.T) {
	scenario := &basicScenario{
		centerTileX: 50,
		centerTileY: 75,
		rng:         rand.New(rand.NewSource(1)),
	}

	got := scenario.staticTiles()
	if len(got) != basicStaticObjectCount {
		t.Fatalf("staticTiles length = %d, want %d", len(got), basicStaticObjectCount)
	}

	seen := make(map[basicTileAnchor]struct{}, len(got))
	runnerTiles := make(map[basicTileAnchor]struct{}, len(scenario.runnerTiles()))
	for _, runnerTile := range scenario.runnerTiles() {
		runnerTiles[runnerTile] = struct{}{}
	}

	for _, tile := range got {
		if _, duplicate := seen[tile]; duplicate {
			t.Fatalf("staticTiles contains duplicate tile %+v", tile)
		}
		if _, overlapsRunner := runnerTiles[tile]; overlapsRunner {
			t.Fatalf("staticTiles overlaps runner tile %+v", tile)
		}

		seen[tile] = struct{}{}
	}
}
