package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/endless/internal/new/camera"
	"github.com/unng-lab/endless/internal/new/tilemap"
)

// TileMapRenderer draws tile maps using ebiten.
type TileMapRenderer struct {
	tiles  *tilemap.TileMap
	colors map[bool]color.Color
	images map[bool]*ebiten.Image
}

// NewTileMapRenderer creates a renderer for the provided tile map.
func NewTileMapRenderer(m *tilemap.TileMap) *TileMapRenderer {
	colors := map[bool]color.Color{
		true:  color.RGBA{R: 65, G: 105, B: 225, A: 255},
		false: color.RGBA{R: 34, G: 139, B: 34, A: 255},
	}

	return &TileMapRenderer{
		tiles:  m,
		colors: colors,
		images: make(map[bool]*ebiten.Image),
	}
}

func (r *TileMapRenderer) tileImage(value bool) *ebiten.Image {
	if img, ok := r.images[value]; ok {
		return img
	}

	img := ebiten.NewImage(1, 1)
	img.Fill(r.colors[value])
	r.images[value] = img
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

	for y := visible.Min.Y; y < visible.Max.Y; y++ {
		for x := visible.Min.X; x < visible.Max.X; x++ {
			img := r.tileImage(r.tiles.TileAt(x, y))

			var op ebiten.DrawImageOptions
			op.GeoM.Scale(tileScreenSize, tileScreenSize)
			op.GeoM.Translate((float64(x)*tileSize-camPos.X)*camScale, (float64(y)*tileSize-camPos.Y)*camScale)

			screen.DrawImage(img, &op)
		}
	}
}
