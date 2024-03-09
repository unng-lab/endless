package endless

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

var B Board

const tileCount = 1024

type Board struct {
	Cells [tileCount][tileCount]Cell
}

func NewBoard() error {
	NewTiles()
	for i := range B.Cells {
		for j := range B.Cells[i] {
			B.Cells[i][j] = Cell{
				tileImage: Tiles[rand.Intn(len(Tiles))],
			}
		}
	}
	return nil
}

func (b *Board) Draw(screen *ebiten.Image) {
	tileSize := TileSize * math.Pow(1.01, float64(C.zoomFactor))
	maxX, maxY := float64(W.GetWidth())/tileSize+1, float64(W.GetHeight())/tileSize+1
	op := &ebiten.DrawImageOptions{}
	dX := C.position[0] / tileSize
	dY := C.position[1] / tileSize
	for j := float64(-1); j < maxY; j++ {
		for i := float64(-1); i < maxX; i++ {
			op.GeoM.Translate(i*TileSize, j*TileSize)
			//op.GeoM.Translate(W.ViewPortCenter(false))
			op.GeoM.Scale(
				math.Pow(1.01, float64(C.zoomFactor)),
				math.Pow(1.01, float64(C.zoomFactor)),
			)
			if posY := tileCount/2 + j + dY; posY < tileCount && posY >= 0 {
				if posX := tileCount/2 + i + dX; posX < tileCount && posX >= 0 {
					screen.DrawImage(b.Cells[int(posY)][int(posX)].tileImage, op)
				}
			}
			op.GeoM.Reset()
		}
	}
}
