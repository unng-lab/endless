package unit

import (
	"fmt"
	"math"

	"github.com/unng-lab/endless/pkg/geom"
)

const (
	projectileSpeed       = 320.0
	projectileDamage      = 1
	projectileRadiusScale = 0.16
	projectileSpawnScale  = 0.48
	projectileRangeTiles  = 14.0
	unitHitRadiusScale    = 0.22
	impactDuration        = 0.18
	impactRadiusScale     = 0.55
)

type projectile struct {
	Position       geom.Point
	Velocity       geom.Point
	OwnerID        int64
	Radius         float64
	Damage         int
	RemainingRange float64
}

type impactEffect struct {
	Position geom.Point
	Radius   float64
	Age      float64
	Duration float64
}

func newProjectile(owner Unit, target geom.Point, tileSize float64) (projectile, error) {
	dx := target.X - owner.Position.X
	dy := target.Y - owner.Position.Y
	length := math.Hypot(dx, dy)
	if length <= 1e-6 {
		return projectile{}, fmt.Errorf("cursor is too close to the unit")
	}

	direction := geom.Point{
		X: dx / length,
		Y: dy / length,
	}
	spawnOffset := tileSize * projectileSpawnScale
	return projectile{
		Position: geom.Point{
			X: owner.Position.X + direction.X*spawnOffset,
			Y: owner.Position.Y + direction.Y*spawnOffset,
		},
		Velocity: geom.Point{
			X: direction.X * projectileSpeed,
			Y: direction.Y * projectileSpeed,
		},
		OwnerID:        owner.ID,
		Radius:         tileSize * projectileRadiusScale,
		Damage:         projectileDamage,
		RemainingRange: tileSize * projectileRangeTiles,
	}, nil
}

func newImpactEffect(position geom.Point, tileSize float64) impactEffect {
	return impactEffect{
		Position: position,
		Radius:   tileSize * impactRadiusScale,
		Duration: impactDuration,
	}
}

func segmentPointIntersection(start, end, center geom.Point, hitRadius float64) (float64, bool) {
	dx := end.X - start.X
	dy := end.Y - start.Y
	lengthSq := dx*dx + dy*dy
	if lengthSq <= 1e-9 {
		if math.Hypot(start.X-center.X, start.Y-center.Y) <= hitRadius {
			return 0, true
		}
		return 0, false
	}

	t := ((center.X-start.X)*dx + (center.Y-start.Y)*dy) / lengthSq
	t = geom.ClampFloat(t, 0, 1)

	closest := pointAlongSegment(start, end, t)
	if math.Hypot(closest.X-center.X, closest.Y-center.Y) <= hitRadius {
		return t, true
	}

	return 0, false
}

func pointAlongSegment(start, end geom.Point, t float64) geom.Point {
	return geom.Point{
		X: start.X + (end.X-start.X)*t,
		Y: start.Y + (end.Y-start.Y)*t,
	}
}
