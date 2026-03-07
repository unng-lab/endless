package render

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/internal/new/assets"
	"github.com/unng-lab/endless/internal/new/camera"
	"github.com/unng-lab/endless/internal/new/unit"
)

// UnitRenderer draws animated units on top of the tile map.
type UnitRenderer struct {
	atlas    *assets.TileAtlas
	tileSize float64
}

// NewUnitRenderer creates a renderer using the provided atlas.
func NewUnitRenderer(atlas *assets.TileAtlas, tileSize float64) *UnitRenderer {
	return &UnitRenderer{atlas: atlas, tileSize: tileSize}
}

// Draw renders the supplied units with respect to the active camera.
func (r *UnitRenderer) Draw(screen *ebiten.Image, cam *camera.Camera, units []*unit.Unit) {
	if len(units) == 0 {
		return
	}

	camPos := cam.Position()
	camScale := cam.Scale()
	tileScreenSize := r.tileSize * camScale
	quality := r.atlas.QualityForScreenSize(tileScreenSize)

	for _, u := range units {
		idx := u.FrameIndex()
		img, err := r.atlas.TileImage(idx, quality)
		if err != nil {
			log.Printf("unit frame %d: %v", idx, err)
			continue
		}

		scale := tileScreenSize / float64(img.Bounds().Dx())
		width := float64(img.Bounds().Dx()) * scale
		height := float64(img.Bounds().Dy()) * scale

		var op ebiten.DrawImageOptions
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate((u.Position.X-camPos.X)*camScale-width/2, (u.Position.Y-camPos.Y)*camScale-height/2)

		screen.DrawImage(img, &op)
	}
}
