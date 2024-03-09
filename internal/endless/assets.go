package endless

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github/unng-lab/madfarmer/assets/img"
)

const (
	bigTileZero int = iota
	bigTileOne
	bigTileTwo
	bigTileThree
	bigTileFour
	bigTileFive
	bigTileSix
	bigTileSeven
	bigTileEight
	bigTileNine
	bigTileTen
	bigTileEleven
	bigTileTwelve
	bigTileThirteen
	bigTileFourteen
	bigTileFifteen
	bigTileSixteen
	bigTileSeventeen
	bigTileEighteen
	bigTileNineteen
	bigTileTwenty
	bigTileTwentyOne
	bigTileTwentyTwo
	bigTileTwentyThree
	bigTileTwentyFour
	bigTileTwentyFive
	bigTileTwentySix
)

var Tiles [256]*ebiten.Image

const (
	TileSize = 16
)

func NewTiles() {
	sprite, err := img.Img("normal.png", 256, 256)
	if err != nil {
		panic(err)
	}
	for j := range 16 {
		for i := range 16 {
			Tiles[j*16+i] = sprite.SubImage(image.Rect(i*TileSize, j*TileSize, (i+1)*TileSize, (j+1)*TileSize)).(*ebiten.Image)
		}
	}
}
