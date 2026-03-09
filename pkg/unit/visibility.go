package unit

import (
	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
)

func UpdateOnScreen(cam *camera.Camera, worldTileSize float64, screenWidth, screenHeight int, unit Unit) {
	if unit == nil {
		return
	}

	base := unit.Base()
	if cam == nil || worldTileSize <= 0 || screenWidth <= 0 || screenHeight <= 0 {
		base.OnScreen = false
		return
	}

	screenRect := geom.Rect{
		Min: geom.Point{},
		Max: geom.Point{X: float64(screenWidth), Y: float64(screenHeight)},
	}
	bounds, ok := unitScreenRect(cam, worldTileSize, unit)
	if !ok {
		base.OnScreen = false
		return
	}

	base.OnScreen = geom.RectsIntersect(bounds, screenRect)
}
