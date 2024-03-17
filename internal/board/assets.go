package board

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/assets/img"
)

var Tiles [256]Tile

const (
	TileSize      = 16
	SmallTileSize = 16
)

func NewTiles() {
	sprite, err := img.Img("normal.png", 256, 256)
	if err != nil {
		panic(err)
	}
	spriteSmall, err := img.Img("small.png", 256, 256)
	if err != nil {
		panic(err)
	}
	for j := range 16 {
		for i := range 16 {
			Tiles[j*16+i].Normal = sprite.SubImage(image.Rect(
				i*TileSize,
				j*TileSize,
				(i+1)*TileSize,
				(j+1)*TileSize,
			)).(*ebiten.Image)
			Tiles[j*16+i].Small = spriteSmall.SubImage(image.Rect(
				i*SmallTileSize,
				j*SmallTileSize,
				(i+1)*SmallTileSize,
				(j+1)*SmallTileSize,
			)).(*ebiten.Image)
		}
	}
}
