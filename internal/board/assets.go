package board

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/unng-lab/madfarmer/assets/img"
)

var Tiles [256]Tile

func NewTiles(tileSize uint64, smallTileSize uint64) {
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
				i*int(tileSize),
				j*int(tileSize),
				(i+1)*int(tileSize),
				(j+1)*int(tileSize),
			)).(*ebiten.Image)
			Tiles[j*16+i].Small = spriteSmall.SubImage(image.Rect(
				i*int(smallTileSize),
				j*int(smallTileSize),
				(i+1)*int(smallTileSize),
				(j+1)*int(smallTileSize),
			)).(*ebiten.Image)
		}
	}
}
