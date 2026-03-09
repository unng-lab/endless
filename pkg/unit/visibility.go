package unit

import (
	"github.com/unng-lab/endless/pkg/camera"
	"github.com/unng-lab/endless/pkg/geom"
)

func UpdateOnScreen(cam *camera.Camera, worldTileSize float64, screenWidth, screenHeight int, u *Unit) {
	if u == nil {
		return
	}
	if cam == nil || worldTileSize <= 0 || screenWidth <= 0 || screenHeight <= 0 {
		u.OnScreen = false
		return
	}

	screenRect := geom.Rect{
		Min: geom.Point{},
		Max: geom.Point{X: float64(screenWidth), Y: float64(screenHeight)},
	}
	u.OnScreen = geom.RectsIntersect(ScreenRect(cam, worldTileSize, *u), screenRect)
}
