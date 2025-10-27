package render

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/internal/new/assets"
	"github.com/unng-lab/endless/internal/new/camera"
	"github.com/unng-lab/endless/internal/new/tilemap"
)

// TileMapRenderer draws tile maps using ebiten.
type TileMapRenderer struct {
	tiles        *tilemap.TileMap
	atlas        *assets.TileAtlas
	tileVariants map[bool][]int
}

// NewTileMapRenderer creates a renderer for the provided tile map.
func NewTileMapRenderer(m *tilemap.TileMap) *TileMapRenderer {
	return &TileMapRenderer{
		tiles: m,
		atlas: assets.NewTileAtlas(),
		tileVariants: map[bool][]int{
			false: {0, 1, 16, 17},
			true:  {32, 33, 48, 49},
		},
	}
}

// Atlas exposes the sprite atlas used by the renderer.
func (r *TileMapRenderer) Atlas() *assets.TileAtlas {
	return r.atlas
}

func (r *TileMapRenderer) tileImage(index int, quality assets.Quality) *ebiten.Image {
	img, err := r.atlas.TileImage(index, quality)
	if err != nil {
		log.Printf("failed to get tile image: %v", err)
		return nil
	}
	return img
}

// Draw renders the visible portion of the map.
func (r *TileMapRenderer) Draw(screen *ebiten.Image, cam *camera.Camera) {
	bounds := screen.Bounds()
	visible := r.tiles.VisibleRange(cam, bounds.Dx(), bounds.Dy())
	camPos := cam.Position()
	camScale := cam.Scale()
	tileSize := r.tiles.TileSize()
	tileScreenSize := tileSize * camScale
	quality := r.atlas.QualityForScreenSize(tileScreenSize)
	columns := r.tiles.Columns()
	if columns == 0 {
		return
	}
	for y := visible.Min.Y; y < visible.Max.Y; y++ {
		for x := visible.Min.X; x < visible.Max.X; x++ {
			value := r.tiles.TileAt(x, y)
			variants, ok := r.tileVariants[value]
			if !ok || len(variants) == 0 {
				continue
			}
			index := variants[(x+y*columns)%len(variants)]
			img := r.tileImage(index, quality)
			if img == nil {
				continue
			}

			scale := tileScreenSize / float64(img.Bounds().Dx())

			var op ebiten.DrawImageOptions
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate((float64(x)*tileSize-camPos.X)*camScale, (float64(y)*tileSize-camPos.Y)*camScale)

			screen.DrawImage(img, &op)
		}
	}
}
