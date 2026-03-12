package scenario

import "github.com/unng-lab/endless/pkg/geom"

// cellAnchor converts one logical tile coordinate into the world-space point at the tile
// center. Scenario seeders use the same helper so mobile and static units start aligned to the
// exact tile grid regardless of which scenario created them.
func cellAnchor(tileX, tileY int, tileSize float64) geom.Point {
	return geom.Point{
		X: (float64(tileX) + 0.5) * tileSize,
		Y: (float64(tileY) + 0.5) * tileSize,
	}
}
