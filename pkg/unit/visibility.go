package unit

import (
	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
)

// unitVisibleOnScreen evaluates visibility directly from the current camera state instead of
// caching a mutable OnScreen flag inside each unit. This keeps draw logic, overlays and tests
// in sync even when the camera moves multiple times between updates.
func unitVisibleOnScreen(cam *camera.Camera, worldTileSize float64, screenWidth, screenHeight int, unit Unit) bool {
	if unit == nil || cam == nil || worldTileSize <= 0 || screenWidth <= 0 || screenHeight <= 0 {
		return false
	}

	screenRect := geom.Rect{
		Min: geom.Point{},
		Max: geom.Point{X: float64(screenWidth), Y: float64(screenHeight)},
	}
	bounds, ok := unitScreenRect(cam, worldTileSize, unit)
	if !ok {
		return false
	}

	return geom.RectsIntersect(bounds, screenRect)
}
