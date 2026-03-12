package rl

import (
	"math/rand"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

const (
	DuelScenarioOpen      = "duel_open"
	DuelScenarioWithCover = "duel_with_cover"
)

type duelScenarioLayout struct {
	ShooterSpawn    geom.Point
	TargetSpawn     geom.Point
	TargetWaypoints []geom.Point
	StaticUnits     []unit.Unit
}

func normalizedDuelScenarioName(name string) string {
	switch name {
	case "", DuelScenarioOpen:
		return DuelScenarioOpen
	case DuelScenarioWithCover:
		return DuelScenarioWithCover
	default:
		return DuelScenarioOpen
	}
}

func buildDuelScenarioLayout(rng *rand.Rand, scenarioName string, gameWorld world.World) duelScenarioLayout {
	switch normalizedDuelScenarioName(scenarioName) {
	case DuelScenarioWithCover:
		return duelCoverLayout(rng, gameWorld)
	case DuelScenarioOpen:
		fallthrough
	default:
		return duelOpenLayout(rng, gameWorld)
	}
}

func duelOpenLayout(rng *rand.Rand, gameWorld world.World) duelScenarioLayout {
	centerTileX := gameWorld.Columns() / 2
	centerTileY := gameWorld.Rows() / 2
	targetOffsetY := 0
	if rng != nil {
		targetOffsetY = rng.Intn(7) - 3
	}

	shooterTileX := centerTileX - 10
	shooterTileY := centerTileY
	targetTileX := centerTileX + 8
	targetTileY := centerTileY + targetOffsetY

	topWaypoint := cellAnchor(targetTileX, targetTileY-5, gameWorld.TileSize())
	bottomWaypoint := cellAnchor(targetTileX, targetTileY+5, gameWorld.TileSize())
	return duelScenarioLayout{
		ShooterSpawn:    cellAnchor(shooterTileX, shooterTileY, gameWorld.TileSize()),
		TargetSpawn:     cellAnchor(targetTileX, targetTileY, gameWorld.TileSize()),
		TargetWaypoints: []geom.Point{topWaypoint, bottomWaypoint},
	}
}

func duelCoverLayout(rng *rand.Rand, gameWorld world.World) duelScenarioLayout {
	centerTileX := gameWorld.Columns() / 2
	centerTileY := gameWorld.Rows() / 2
	shooterTileX := centerTileX - 3
	targetTileX := centerTileX + 7
	targetOffsetY := 0
	if rng != nil {
		targetOffsetY = rng.Intn(3) - 1
	}

	targetTileY := centerTileY + targetOffsetY
	staticUnits := make([]unit.Unit, 0, 8)

	// The central staggered barricade line forces both pathfinding and projectile trajectories
	// to route around narrow windows instead of taking a trivial straight-line duel.
	coverTiles := [][3]int{
		{centerTileX - 1, centerTileY - 3, 0},
		{centerTileX - 1, centerTileY - 2, 1},
		{centerTileX - 1, centerTileY, 1},
		{centerTileX - 1, centerTileY + 2, 1},
		{centerTileX - 1, centerTileY + 3, 0},
		{centerTileX + 1, centerTileY - 2, 0},
		{centerTileX + 1, centerTileY, 0},
		{centerTileX + 1, centerTileY + 2, 0},
	}
	for _, coverTile := range coverTiles {
		position := cellAnchor(coverTile[0], coverTile[1], gameWorld.TileSize())
		if coverTile[2] == 0 {
			staticUnits = append(staticUnits, unit.NewWall(position))
			continue
		}
		staticUnits = append(staticUnits, unit.NewBarricade(position))
	}

	targetWaypoints := []geom.Point{
		cellAnchor(targetTileX, centerTileY-4, gameWorld.TileSize()),
		cellAnchor(targetTileX-1, centerTileY+4, gameWorld.TileSize()),
		cellAnchor(targetTileX, centerTileY+1, gameWorld.TileSize()),
	}
	return duelScenarioLayout{
		ShooterSpawn:    cellAnchor(shooterTileX, centerTileY, gameWorld.TileSize()),
		TargetSpawn:     cellAnchor(targetTileX, targetTileY, gameWorld.TileSize()),
		TargetWaypoints: targetWaypoints,
		StaticUnits:     staticUnits,
	}
}
