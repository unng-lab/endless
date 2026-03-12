package rl

import (
	"math"

	"github.com/unng-lab/endless/pkg/geom"
	"github.com/unng-lab/endless/pkg/unit"
	"github.com/unng-lab/endless/pkg/world"
)

const (
	occupancyUnknown         int16 = -1
	occupancyEmpty           int16 = 0
	occupancyShooter         int16 = 1
	occupancyTarget          int16 = 2
	occupancyFriendlyShot    int16 = 3
	occupancyHostileShot     int16 = 4
	occupancyMovementBlocker int16 = 5
)

func buildObservation(
	gameWorld world.World,
	snapshot unit.DuelSnapshot,
	projectiles []unit.ProjectileSnapshot,
	blockers []unit.BlockingUnitSnapshot,
	previousTargetPos geom.Point,
	hasPreviousTargetPos bool,
	recentMoveFailure bool,
) Observation {
	terrainPatch, occupancyPatch := buildLocalTilePatches(gameWorld, snapshot, projectiles, blockers, duelObservationPatchRadius)
	friendlyShot, hostileShot := buildNearestProjectileFeatures(snapshot, projectiles)
	hasDestination, destinationRelativeX, destinationRelativeY, distanceToDestination := buildDestinationFeatures(snapshot.Shooter)

	return Observation{
		Snapshot:                     snapshot,
		PreviousTargetPos:            previousTargetPos,
		HasPreviousTargetPos:         hasPreviousTargetPos,
		ShooterHasDestination:        hasDestination,
		ShooterDestinationRelativeX:  destinationRelativeX,
		ShooterDestinationRelativeY:  destinationRelativeY,
		ShooterDistanceToDestination: distanceToDestination,
		ShooterRecentMoveFailure:     recentMoveFailure,
		PatchRadius:                  duelObservationPatchRadius,
		LocalTerrainPatch:            terrainPatch,
		LocalOccupancyPatch:          occupancyPatch,
		NearestFriendlyShot:          friendlyShot,
		NearestHostileShot:           hostileShot,
		TileSize:                     gameWorld.TileSize(),
		WorldWidth:                   gameWorld.Width(),
		WorldHeight:                  gameWorld.Height(),
	}
}

func buildLocalTilePatches(gameWorld world.World, snapshot unit.DuelSnapshot, projectiles []unit.ProjectileSnapshot, blockers []unit.BlockingUnitSnapshot, radius int) ([]int16, []int16) {
	patchWidth := radius*2 + 1
	patchArea := patchWidth * patchWidth
	terrainPatch := make([]int16, 0, patchArea)
	occupancyPatch := make([]int16, 0, patchArea)
	projectileOccupancy := projectileOccupancyByTile(gameWorld, snapshot.Shooter.UnitID, projectiles)
	blockerOccupancy := blockerOccupancyByTile(blockers)

	for offsetY := -radius; offsetY <= radius; offsetY++ {
		for offsetX := -radius; offsetX <= radius; offsetX++ {
			tileX := snapshot.Shooter.TileX + offsetX
			tileY := snapshot.Shooter.TileY + offsetY
			if !gameWorld.InBounds(tileX, tileY) {
				terrainPatch = append(terrainPatch, -1)
				occupancyPatch = append(occupancyPatch, occupancyUnknown)
				continue
			}

			terrainPatch = append(terrainPatch, int16(gameWorld.TileType(tileX, tileY)))
			occupancyPatch = append(occupancyPatch, occupancyCodeForTile(snapshot, tileX, tileY, projectileOccupancy, blockerOccupancy))
		}
	}

	return terrainPatch, occupancyPatch
}

func projectileOccupancyByTile(gameWorld world.World, shooterID int64, projectiles []unit.ProjectileSnapshot) map[[2]int]int16 {
	if len(projectiles) == 0 {
		return nil
	}

	occupancy := make(map[[2]int]int16, len(projectiles))
	for _, projectile := range projectiles {
		if projectile.Exploding {
			continue
		}

		tileX, tileY, ok := worldPointToTile(gameWorld, projectile.Position)
		if !ok {
			continue
		}

		code := occupancyHostileShot
		if projectile.OwnerID == shooterID {
			code = occupancyFriendlyShot
		}
		key := [2]int{tileX, tileY}
		existing, exists := occupancy[key]
		if !exists || existing == occupancyFriendlyShot && code == occupancyHostileShot {
			occupancy[key] = code
		}
	}

	return occupancy
}

func blockerOccupancyByTile(blockers []unit.BlockingUnitSnapshot) map[[2]int]int16 {
	if len(blockers) == 0 {
		return nil
	}

	occupancy := make(map[[2]int]int16, len(blockers))
	for _, blocker := range blockers {
		occupancy[[2]int{blocker.TileX, blocker.TileY}] = occupancyMovementBlocker
	}
	return occupancy
}

func occupancyCodeForTile(snapshot unit.DuelSnapshot, tileX, tileY int, projectileOccupancy, blockerOccupancy map[[2]int]int16) int16 {
	switch {
	case snapshot.Shooter.TileX == tileX && snapshot.Shooter.TileY == tileY:
		return occupancyShooter
	case snapshot.Target.TileX == tileX && snapshot.Target.TileY == tileY:
		return occupancyTarget
	}
	if blockerOccupancy != nil {
		if code, ok := blockerOccupancy[[2]int{tileX, tileY}]; ok {
			return code
		}
	}

	if projectileOccupancy == nil {
		return occupancyEmpty
	}
	if code, ok := projectileOccupancy[[2]int{tileX, tileY}]; ok {
		return code
	}
	return occupancyEmpty
}

func buildNearestProjectileFeatures(snapshot unit.DuelSnapshot, projectiles []unit.ProjectileSnapshot) (ProjectileFeature, ProjectileFeature) {
	friendly := ProjectileFeature{}
	hostile := ProjectileFeature{}
	bestFriendlyDistance := math.Inf(1)
	bestHostileDistance := math.Inf(1)

	for _, projectile := range projectiles {
		if projectile.Exploding {
			continue
		}

		relative := geom.Point{
			X: projectile.Position.X - snapshot.Shooter.Position.X,
			Y: projectile.Position.Y - snapshot.Shooter.Position.Y,
		}
		distance := math.Hypot(relative.X, relative.Y)
		if distance <= 1e-6 {
			continue
		}

		feature := ProjectileFeature{
			Exists:    true,
			RelativeX: relative.X,
			RelativeY: relative.Y,
			Distance:  distance,
		}
		if projectile.OwnerID == snapshot.Shooter.UnitID {
			if distance < bestFriendlyDistance {
				bestFriendlyDistance = distance
				friendly = feature
			}
			continue
		}
		if distance < bestHostileDistance {
			bestHostileDistance = distance
			hostile = feature
		}
	}

	return friendly, hostile
}

func buildDestinationFeatures(shooter unit.UnitSnapshot) (bool, float64, float64, float64) {
	if !shooter.HasDestination {
		return false, 0, 0, 0
	}

	relativeX := shooter.Destination.X - shooter.Position.X
	relativeY := shooter.Destination.Y - shooter.Position.Y
	return true, relativeX, relativeY, math.Hypot(relativeX, relativeY)
}

func worldPointToTile(gameWorld world.World, position geom.Point) (int, int, bool) {
	if position.X < 0 || position.Y < 0 || position.X >= gameWorld.Width() || position.Y >= gameWorld.Height() {
		return 0, 0, false
	}

	tileX := int(math.Floor(position.X / gameWorld.TileSize()))
	tileY := int(math.Floor(position.Y / gameWorld.TileSize()))
	if !gameWorld.InBounds(tileX, tileY) {
		return 0, 0, false
	}

	return tileX, tileY, true
}
